package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"rms-backend/internal/repository"
)

type PRPaymentExportFilter struct {
	StartDate     string
	EndDate       string
	ResponsibleID int
	Status        string
	AdminID       string
}

type PRPaymentExportService struct {
	paymentRepo *repository.PRPaymentRepo
	prRepo      *repository.PRRepo
}

func NewPRPaymentExportService(paymentRepo *repository.PRPaymentRepo, prRepo *repository.PRRepo) *PRPaymentExportService {
	return &PRPaymentExportService{paymentRepo: paymentRepo, prRepo: prRepo}
}

var exportHeaders = []string{
	"No", "No RFP", "Nama PIC RUAS", "Item Pekerjaan", "Responsible", "COA", "Subclient",
	"Amount Diajukan", "Tujuan Rekening - Nama", "Tujuan Rekening - No Rekening",
	"Total Pembayaran Sebelumnya", "Amount PO", "Amount HPP", "Margin", "%",
	"NO. PO", "Due Date Invoicing", "Status Pembayaran (TTD PIC)",
}

func (s *PRPaymentExportService) GenerateXlsx(ctx context.Context, f PRPaymentExportFilter) ([]byte, string, error) {
	rows, err := s.paymentRepo.ListForExport(ctx, repository.PaymentExportFilter{
		StartDate: f.StartDate, EndDate: f.EndDate,
		ResponsibleID: f.ResponsibleID, Status: f.Status, AdminID: f.AdminID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("query payment export: %w", err)
	}

	file := excelize.NewFile()
	sheet := file.GetSheetName(0)

	boldStyle, _ := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	numStyle, _ := file.NewStyle(&excelize.Style{NumFmt: 3})
	pctStyle, _ := file.NewStyle(&excelize.Style{NumFmt: 10})

	for i, h := range exportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = file.SetCellValue(sheet, cell, h)
		_ = file.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	if len(rows) == 0 {
		noteCell, _ := excelize.CoordinatesToCellName(1, 2)
		_ = file.SetCellValue(sheet, noteCell, "Tidak ada data payment yang cocok dengan filter yang dipilih.")
		buf, err := file.WriteToBuffer()
		if err != nil {
			return nil, "", fmt.Errorf("write buffer: %w", err)
		}
		filename := fmt.Sprintf("Summary_Payment_PR_%s.xlsx", time.Now().Format("20060102_150405"))
		return buf.Bytes(), filename, nil
	}

	type subtotalAgg struct {
		Name  string
		Total float64
	}
	subtotals := map[string]*subtotalAgg{}
	var subtotalOrder []string
	grandTotal := 0.0

	rowIdx := 2
	for i, row := range rows {
		prevTotal, err := s.previousPaidTotal(ctx, row)
		if err != nil {
			return nil, "", fmt.Errorf("hitung total pembayaran sebelumnya (payment_id=%d): %w", row.PaymentID, err)
		}

		margin := row.PoAmount - row.Hpp
		marginPct := 0.0
		if row.PoAmount != 0 {
			marginPct = margin / row.PoAmount
		}

		poNo := row.PoNo
		if poNo == "" {
			poNo = "PO BELUM RELEASE"
		}
		dueDate := ""
		if row.TargetInvoiceDate != nil {
			dueDate = row.TargetInvoiceDate.Format("2006-01-02")
		}

		cells := map[int]interface{}{
			1: i + 1, 2: row.RfpNo, 3: row.RequesterName, 4: row.DescriptionItem,
			5: row.ResponsibleName, 6: row.CoaCode, 7: row.SubClient, 8: row.Amount,
			9: row.BankAccountName, 10: row.BankAccountNo,
			12: row.PoAmount, 13: row.Hpp, 14: margin, 15: marginPct,
			16: poNo, 17: dueDate, 18: statusDisplayLabel(row.PaymentStatus),
		}
		if prevTotal != 0 {
			cells[11] = prevTotal // kosongkan kalau memang tidak ada pembayaran sebelumnya
		}
		for col, v := range cells {
			cell, _ := excelize.CoordinatesToCellName(col, rowIdx)
			_ = file.SetCellValue(sheet, cell, v)
		}
		for _, col := range []int{8, 11, 12, 13, 14} {
			cell, _ := excelize.CoordinatesToCellName(col, rowIdx)
			_ = file.SetCellStyle(sheet, cell, cell, numStyle)
		}
		pctCell, _ := excelize.CoordinatesToCellName(15, rowIdx)
		_ = file.SetCellStyle(sheet, pctCell, pctCell, pctStyle)

		if row.PaymentStatus == "paid" {
			key := row.ResponsibleName
			if _, ok := subtotals[key]; !ok {
				subtotals[key] = &subtotalAgg{Name: row.ResponsibleName}
				subtotalOrder = append(subtotalOrder, key)
			}
			subtotals[key].Total += row.Amount
			grandTotal += row.Amount
		}
		rowIdx++
	}

	// baris subtotal per responsible (mengikuti pola "Jumlah Payment <Responsible>"
	// di dokumen referensi) - hanya menghitung payment berstatus 'paid'.
	rowIdx++
	for _, key := range subtotalOrder {
		agg := subtotals[key]
		labelCell, _ := excelize.CoordinatesToCellName(4, rowIdx)
		_ = file.SetCellValue(sheet, labelCell, fmt.Sprintf("Jumlah Payment %s", agg.Name))
		_ = file.SetCellStyle(sheet, labelCell, labelCell, boldStyle)
		valCell, _ := excelize.CoordinatesToCellName(5, rowIdx)
		_ = file.SetCellValue(sheet, valCell, agg.Total)
		_ = file.SetCellStyle(sheet, valCell, valCell, numStyle)
		rowIdx++
	}
	grandLabelCell, _ := excelize.CoordinatesToCellName(4, rowIdx)
	_ = file.SetCellValue(sheet, grandLabelCell, "Jumlah Payment")
	_ = file.SetCellStyle(sheet, grandLabelCell, grandLabelCell, boldStyle)
	grandValCell, _ := excelize.CoordinatesToCellName(5, rowIdx)
	_ = file.SetCellValue(sheet, grandValCell, grandTotal)
	_ = file.SetCellStyle(sheet, grandValCell, grandValCell, numStyle)

	for i := range exportHeaders {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		_ = file.SetColWidth(sheet, colLetter, colLetter, 18)
	}
	_ = file.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	buf, err := file.WriteToBuffer()
	if err != nil {
		return nil, "", fmt.Errorf("write buffer: %w", err)
	}

	filename := fmt.Sprintf("Summary_Payment_PR_%s.xlsx", time.Now().Format("20060102_150405"))
	return buf.Bytes(), filename, nil
}

func (s *PRPaymentExportService) previousPaidTotal(ctx context.Context, row repository.PaymentExportRow) (float64, error) {
	chain, err := s.prRepo.GetReferenceChain(ctx, row.PRID)
	if err != nil {
		return 0, err
	}
	ids := make([]int, 0, len(chain))
	for _, pr := range chain {
		ids = append(ids, pr.ID)
	}
	return s.paymentRepo.SumPaidBeforeInChain(ctx, ids, row.PaymentCreateDate, row.PaymentID)
}

func statusDisplayLabel(status string) string {
	switch status {
	case "paid":
		return "DONE"
	case "cancelled":
		return "CANCELLED"
	case "draft":
		return "DRAFT (PR belum approved)"
	default:
		return "-"
	}
}