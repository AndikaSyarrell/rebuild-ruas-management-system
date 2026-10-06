package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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

// GenerateXlsx menghasilkan workbook "Summary List Payment": satu sheet per tanggal
// pembuatan payment (layout di pr_payment_export_layout.go).
func (s *PRPaymentExportService) GenerateXlsx(ctx context.Context, f PRPaymentExportFilter) ([]byte, string, error) {
	rows, err := s.paymentRepo.ListForExport(ctx, repository.PaymentExportFilter{
		StartDate: f.StartDate, EndDate: f.EndDate,
		ResponsibleID: f.ResponsibleID, Status: f.Status, AdminID: f.AdminID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("query payment export: %w", err)
	}

	summary := make([]PaymentSummaryRow, 0, len(rows))
	for _, row := range rows {
		prev, err := s.previousPaidTotal(ctx, row)
		if err != nil {
			return nil, "", fmt.Errorf("hitung total pembayaran sebelumnya (payment_id=%d): %w", row.PaymentID, err)
		}
		summary = append(summary, PaymentSummaryRow{PaymentExportRow: row, PrevPaid: prev})
	}

	file, err := BuildPaymentSummaryWorkbook(summary)
	if err != nil {
		return nil, "", fmt.Errorf("build workbook: %w", err)
	}
	defer file.Close()

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

// ---------------------------------------------------------------------
// Layout workbook "Summary List Payment"
// ---------------------------------------------------------------------

// Layout export "Summary List Payment": satu sheet per tanggal payment, meniru
// file SUMMARY_LIST_PAYMENT_MA_2026.xlsx (blok info di kiri atas, header dua
// baris dengan sel gabungan, subtotal per responsible, blok tanda tangan).

const paymentPoNotReleased = "PO BELUM RELEASE"

var indonesianMonths = [...]string{"", "JANUARI", "FEBRUARI", "MARET", "APRIL", "MEI", "JUNI",
	"JULI", "AGUSTUS", "SEPTEMBER", "OKTOBER", "NOVEMBER", "DESEMBER"}

// Kolom A..R (18 kolom). Lebar mengikuti sheet contoh; kolom E sedikit dilebarkan
// supaya angka subtotal berfont 20pt tidak tampil "####".
var paymentColWidths = [...]float64{11.4, 27.2, 22.5, 52.9, 26, 15.2, 15.8, 17.1, 22.8, 27.4, 21, 19, 16.6, 16.9, 15.4, 28.6, 23, 17.9}

const (
	psFirstDataRow   = 6
	psMinDataRowH    = 60.0
	psSubtotalRowH   = 40.0
	psAccountingFmt  = `_(* #,##0_);_(* \(#,##0\);_(* "-"_);_(@_)`
	psRupiahFmt      = `[$Rp-421]#,##0`
	psDueDateFmt     = "d mmm"
	psHeaderFillRGB  = "8496B0"
	psTotalFillRGB   = "FFFF00"
	psStatusDone     = "DONE"
	psStatusCanceled = "CANCELLED"
	psStatusDraft    = "DRAFT"
)

// PaymentSummaryRow = satu baris payment + total pembayaran sebelumnya (rantai PR).
type PaymentSummaryRow struct {
	repository.PaymentExportRow
	PrevPaid float64
}

// periodWeek: minggu ke-N dalam bulan, minggu dimulai Senin, minggu parsial di awal
// bulan dihitung minggu 1, maksimal 4 (pola yang dipakai sheet September 2026).
func periodWeek(t time.Time) int {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	offset := (int(first.Weekday()) + 6) % 7 // Senin = 0
	w := (t.Day()-1+offset)/7 + 1
	if w > 4 {
		w = 4
	}
	return w
}

func paymentSheetName(t time.Time) string {
	return fmt.Sprintf("%02d %s %d", t.Day(), indonesianMonths[t.Month()], t.Year())
}

// paymentDueCell menentukan isi kolom "Due Date Invoicing": tanggal target invoice untuk
// payment yang masih berjalan, atau label status untuk payment yang sudah selesai/batal/draft.
func paymentDueCell(r PaymentSummaryRow) interface{} {
	switch r.PaymentStatus {
	case "paid":
		return psStatusDone
	case "cancelled":
		return psStatusCanceled
	case "draft":
		return psStatusDraft
	}
	if r.TargetInvoiceDate != nil {
		d := *r.TargetInvoiceDate
		return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	}
	return ""
}

type psStyles struct {
	label, labelBottom, header     int
	center, left, money, pct, due  int
	subLabel, subValue, totalValue int
	sigBox, sigName, grid          int
}

func newPSStyles(f *excelize.File) (*psStyles, error) {
	border := func() []excelize.Border {
		out := make([]excelize.Border, 0, 4)
		for _, side := range []string{"left", "right", "top", "bottom"} {
			out = append(out, excelize.Border{Type: side, Color: "000000", Style: 1})
		}
		return out
	}
	font := func(size float64, bold bool) *excelize.Font {
		return &excelize.Font{Family: "Calibri", Size: size, Bold: bold, Color: "000000"}
	}
	fill := func(rgb string) excelize.Fill {
		return excelize.Fill{Type: "pattern", Color: []string{rgb}, Pattern: 1}
	}
	align := func(h string) *excelize.Alignment {
		return &excelize.Alignment{Horizontal: h, Vertical: "center", WrapText: true}
	}
	numFmt := func(s string) *string { return &s }

	var err error
	mk := func(s *excelize.Style) int {
		if err != nil {
			return 0
		}
		var id int
		id, err = f.NewStyle(s)
		return id
	}

	st := &psStyles{}
	st.label = mk(&excelize.Style{Font: font(15, false), Alignment: align("left")})
	st.labelBottom = mk(&excelize.Style{Font: font(15, false), Alignment: align("left"),
		Border: []excelize.Border{{Type: "bottom", Color: "000000", Style: 1}}})
	st.header = mk(&excelize.Style{Font: font(15, false), Fill: fill(psHeaderFillRGB), Border: border(), Alignment: align("center")})
	st.center = mk(&excelize.Style{Font: font(15, false), Fill: fill("FFFFFF"), Border: border(), Alignment: align("center")})
	st.left = mk(&excelize.Style{Font: font(15, false), Fill: fill("FFFFFF"), Border: border(), Alignment: align("left")})
	st.money = mk(&excelize.Style{Font: font(15, false), Fill: fill("FFFFFF"), Border: border(), Alignment: align("right"),
		CustomNumFmt: numFmt(psAccountingFmt)})
	st.pct = mk(&excelize.Style{Font: font(15, false), Fill: fill("FFFFFF"), Border: border(), Alignment: align("center"), NumFmt: 9})
	st.due = mk(&excelize.Style{Font: font(15, false), Fill: fill("FFFFFF"), Border: border(), Alignment: align("center"),
		CustomNumFmt: numFmt(psDueDateFmt)})
	st.subLabel = mk(&excelize.Style{Font: font(20, true), Border: border(), Alignment: align("left")})
	st.subValue = mk(&excelize.Style{Font: font(20, true), Border: border(), Alignment: align("center"), NumFmt: 3})
	st.totalValue = mk(&excelize.Style{Font: font(15, true), Fill: fill(psTotalFillRGB), Border: border(), Alignment: align("center"),
		CustomNumFmt: numFmt(psRupiahFmt)})
	st.grid = mk(&excelize.Style{Font: font(15, false), Border: border()})
	st.sigBox = mk(&excelize.Style{Font: font(11, true), Border: border(), Alignment: align("center")})
	st.sigName = mk(&excelize.Style{Font: font(11, true), Border: border(), Alignment: align("center")})
	return st, err
}

// sheetWriter mengumpulkan error pertama supaya kode layout tidak penuh `if err != nil`.
type sheetWriter struct {
	f     *excelize.File
	sheet string
	err   error
}

func (w *sheetWriter) val(cell string, v interface{}) {
	if w.err == nil {
		w.err = w.f.SetCellValue(w.sheet, cell, v)
	}
}

func (w *sheetWriter) formula(cell, formula string) {
	if w.err == nil {
		w.err = w.f.SetCellFormula(w.sheet, cell, formula)
	}
}

func (w *sheetWriter) style(from, to string, id int) {
	if w.err == nil {
		w.err = w.f.SetCellStyle(w.sheet, from, to, id)
	}
}

func (w *sheetWriter) merge(from, to string) {
	if w.err == nil {
		w.err = w.f.MergeCell(w.sheet, from, to)
	}
}

func (w *sheetWriter) height(row int, h float64) {
	if w.err == nil {
		w.err = w.f.SetRowHeight(w.sheet, row, h)
	}
}

func cellRef(col string, row int) string { return fmt.Sprintf("%s%d", col, row) }

// estimateRowHeight memperkirakan tinggi baris dari teks terpanjang yang di-wrap
// (font 15pt: sekitar 0,72 karakter per satuan lebar kolom).
func estimateRowHeight(r PaymentSummaryRow) float64 {
	lines := func(s string, colWidth float64) int {
		perLine := int(math.Max(1, colWidth*0.72))
		n := utf8.RuneCountInString(s)
		return (n + perLine - 1) / perLine
	}
	maxLines := 1
	for _, c := range []struct {
		s string
		w float64
	}{
		{r.DescriptionItem, paymentColWidths[3]},
		{r.RequesterName, paymentColWidths[2]},
		{r.BankAccountName, paymentColWidths[8]},
		{r.SubClient, paymentColWidths[6]},
		{r.PoNo, paymentColWidths[15]},
	} {
		if l := lines(c.s, c.w); l > maxLines {
			maxLines = l
		}
	}
	h := float64(maxLines)*21 + 12
	return math.Min(math.Max(h, psMinDataRowH), 300)
}

func accountDest(r PaymentSummaryRow) string {
	switch {
	case r.Bank != "" && r.BankAccountNo != "":
		return r.Bank + "-" + r.BankAccountNo
	case r.BankAccountNo != "":
		return r.BankAccountNo
	}
	return r.Bank
}

func (st *psStyles) writeDaySheet(f *excelize.File, sheet string, day time.Time, rows []PaymentSummaryRow) error {
	w := &sheetWriter{f: f, sheet: sheet}

	for i, width := range paymentColWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		if w.err == nil {
			w.err = f.SetColWidth(sheet, col, col, width)
		}
	}

	// --- Blok info (A1:B3) ---
	w.val("A1", "Date")
	w.val("B1", ": "+day.Format("02/01/2006"))
	w.val("A2", "Periode Week")
	w.val("B2", fmt.Sprintf(": %02d", periodWeek(day)))
	w.val("A3", "Periode Month")
	w.val("B3", ": "+indonesianMonths[day.Month()])
	w.style("A1", "B2", st.label)
	w.style("A3", "B3", st.labelBottom)
	w.height(1, 27.75)

	// --- Header dua baris (A4:R5) ---
	for _, h := range []struct{ col, text string }{
		{"A", "No"}, {"B", "No RFP"}, {"C", "Nama PIC RUAS"}, {"D", "Item Pekerjaan"},
		{"E", "Responsible"}, {"F", "COA"}, {"G", "Subclient"}, {"H", "Amount Diajukan"},
		{"P", "NO. PO"}, {"Q", "Due Date Invoicing"}, {"R", "TTD PIC"},
	} {
		w.val(cellRef(h.col, 4), h.text)
		w.merge(cellRef(h.col, 4), cellRef(h.col, 5))
	}
	w.val("I4", "Tujuan Rekening")
	w.merge("I4", "J4")
	w.val("I5", "Nama")
	w.val("J5", "No Rekening")
	w.val("K4", "Margin")
	w.merge("K4", "O4")
	for col, text := range map[string]string{
		"K": "Total Pembayaran Sebelumnya", "L": "Amount PO", "M": "Amount HPP", "N": "Margin", "O": "%",
	} {
		w.val(cellRef(col, 5), text)
	}
	w.style("A4", "R5", st.header)
	w.height(4, 24)
	w.height(5, 42)

	// --- Baris data ---
	last := psFirstDataRow + len(rows) - 1
	for i, r := range rows {
		n := psFirstDataRow + i
		w.val(cellRef("A", n), i+1)
		w.val(cellRef("B", n), r.RfpNo)
		w.val(cellRef("C", n), strings.ToUpper(r.RequesterName))
		w.val(cellRef("D", n), strings.ToUpper(r.DescriptionItem))
		w.val(cellRef("E", n), strings.ToUpper(r.ResponsibleName))
		w.val(cellRef("F", n), r.CoaCode)
		w.val(cellRef("G", n), strings.ToUpper(r.SubClient))
		w.val(cellRef("H", n), r.Amount)
		w.val(cellRef("I", n), strings.ToUpper(r.BankAccountName))
		w.val(cellRef("J", n), accountDest(r)) // teks, supaya no rekening panjang tidak berubah jadi notasi ilmiah
		if r.PrevPaid != 0 {
			w.val(cellRef("K", n), r.PrevPaid)
		}
		w.val(cellRef("L", n), r.PoAmount)
		w.val(cellRef("M", n), r.Hpp)
		w.formula(cellRef("N", n), fmt.Sprintf("L%d-M%d", n, n))
		w.formula(cellRef("O", n), fmt.Sprintf(`IF(L%d=0,"",N%d/L%d)`, n, n, n))
		po := strings.TrimSpace(r.PoNo)
		if po == "" {
			po = paymentPoNotReleased
		}
		w.val(cellRef("P", n), po)
		due := paymentDueCell(r)
		w.val(cellRef("Q", n), due)

		w.style(cellRef("A", n), cellRef("C", n), st.center)
		w.style(cellRef("D", n), cellRef("D", n), st.left)
		w.style(cellRef("E", n), cellRef("G", n), st.center)
		w.style(cellRef("H", n), cellRef("H", n), st.money)
		w.style(cellRef("I", n), cellRef("J", n), st.center)
		w.style(cellRef("K", n), cellRef("N", n), st.money)
		w.style(cellRef("O", n), cellRef("O", n), st.pct)
		w.style(cellRef("P", n), cellRef("P", n), st.center)
		if _, isDate := due.(time.Time); isDate {
			w.style(cellRef("Q", n), cellRef("Q", n), st.due)
		} else {
			w.style(cellRef("Q", n), cellRef("Q", n), st.center)
		}
		w.style(cellRef("R", n), cellRef("R", n), st.center) // TTD PIC: dibiarkan kosong untuk tanda tangan basah
		w.height(n, estimateRowHeight(r))
	}

	// --- Subtotal per responsible + total ---
	// Payment CANCELLED dan DRAFT tidak ikut dijumlahkan (dikenali dari label di kolom Q).
	seen := map[string]bool{}
	var responsibles []string
	for _, r := range rows {
		name := strings.ToUpper(strings.TrimSpace(r.ResponsibleName))
		if !seen[name] {
			seen[name] = true
			responsibles = append(responsibles, name)
		}
	}
	sort.Strings(responsibles)

	subStart := last + 1
	for j, name := range responsibles {
		n := subStart + j
		label := "Jumlah Payment " + name
		if name == "" {
			label = "Jumlah Payment (tanpa responsible)"
		}
		w.style(cellRef("A", n), cellRef("O", n), st.grid)
		w.val(cellRef("D", n), label)
		w.formula(cellRef("E", n), fmt.Sprintf(
			`SUMIFS($H$%d:$H$%d,$E$%d:$E$%d,"%s",$Q$%d:$Q$%d,"<>%s",$Q$%d:$Q$%d,"<>%s")`,
			psFirstDataRow, last, psFirstDataRow, last, strings.ReplaceAll(name, `"`, `""`),
			psFirstDataRow, last, psStatusCanceled, psFirstDataRow, last, psStatusDraft))
		w.style(cellRef("D", n), cellRef("D", n), st.subLabel)
		w.style(cellRef("E", n), cellRef("E", n), st.subValue)
		w.height(n, psSubtotalRowH)
	}
	total := subStart + len(responsibles)
	w.style(cellRef("A", total), cellRef("O", total), st.grid)
	w.val(cellRef("D", total), "Jumlah Payment")
	w.formula(cellRef("E", total), fmt.Sprintf("SUM(E%d:E%d)", subStart, total-1))
	w.style(cellRef("D", total), cellRef("D", total), st.subLabel)
	w.style(cellRef("E", total), cellRef("E", total), st.totalValue)
	w.height(total, psSubtotalRowH)

	// --- Blok tanda tangan (P:R): kotak kosong untuk tanda tangan basah, tanpa nama ---
	sigTop := total - 1
	w.style(cellRef("P", sigTop), cellRef("R", sigTop), st.sigBox)
	for _, col := range []string{"P", "Q", "R"} {
		w.merge(cellRef(col, total), cellRef(col, total+1))
		w.style(cellRef(col, total), cellRef(col, total+1), st.sigName)
	}
	w.height(sigTop, psSubtotalRowH+10)

	// --- Pengaturan cetak: A4 landscape, muat 1 halaman lebar ---
	if w.err == nil {
		size, orient, one, zero := 9, "landscape", 1, 0
		w.err = f.SetPageLayout(sheet, &excelize.PageLayoutOptions{Size: &size, Orientation: &orient, FitToWidth: &one, FitToHeight: &zero})
	}
	if w.err == nil {
		fit := true
		w.err = f.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: &fit})
	}
	return w.err
}

// BuildPaymentSummaryWorkbook membangun workbook: satu sheet per tanggal pembuatan payment,
// diurutkan dari tanggal terlama. Tanpa data, menghasilkan satu sheet berisi keterangan.
func BuildPaymentSummaryWorkbook(rows []PaymentSummaryRow) (*excelize.File, error) {
	f := excelize.NewFile()
	if len(rows) == 0 {
		if err := f.SetSheetName("Sheet1", "TIDAK ADA DATA"); err != nil {
			return nil, err
		}
		_ = f.SetCellValue("TIDAK ADA DATA", "A1", "Tidak ada data payment yang cocok dengan filter yang dipilih.")
		return f, nil
	}

	st, err := newPSStyles(f)
	if err != nil {
		return nil, fmt.Errorf("styles: %w", err)
	}

	type dayGroup struct {
		day  time.Time
		rows []PaymentSummaryRow
	}
	groups := map[string]*dayGroup{}
	var keys []string
	for _, r := range rows {
		d := r.PaymentCreateDate
		key := d.Format("2006-01-02")
		g, ok := groups[key]
		if !ok {
			g = &dayGroup{day: time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())}
			groups[key] = g
			keys = append(keys, key)
		}
		g.rows = append(g.rows, r)
	}
	sort.Strings(keys)

	for i, key := range keys {
		g := groups[key]
		name := paymentSheetName(g.day)
		if i == 0 {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				return nil, err
			}
		} else if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
		if err := st.writeDaySheet(f, name, g.day, g.rows); err != nil {
			return nil, fmt.Errorf("sheet %s: %w", name, err)
		}
	}
	f.SetActiveSheet(0)
	return f, nil
}