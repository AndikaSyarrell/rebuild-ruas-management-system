package service

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"rms-backend/internal/repository"
)

var ErrPRExportSourceMissing = errors.New("purchase request tidak ditemukan untuk export RFP")

const defaultCompanyName = "PT. RUAS MANAGEMENT SYSTEM"

type PRExportService struct {
	prRepo        *repository.PRRepo
	approvalRepo  *repository.PRApprovalRepo
	paymentRepo   *repository.PRPaymentRepo
	signatureRepo *repository.AdminSignatureRepo
	uploadDir     string // root penyimpanan file upload, mis. "./storage/uploads"
}

func NewPRExportService(
	prRepo *repository.PRRepo,
	approvalRepo *repository.PRApprovalRepo,
	paymentRepo *repository.PRPaymentRepo,
	signatureRepo *repository.AdminSignatureRepo,
	uploadDir string,
) *PRExportService {
	return &PRExportService{
		prRepo: prRepo, approvalRepo: approvalRepo, paymentRepo: paymentRepo,
		signatureRepo: signatureRepo, uploadDir: uploadDir,
	}
}

func (s *PRExportService) GenerateRFPDocx(ctx context.Context, prID int) ([]byte, string, error) {
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrPRExportSourceMissing
		}
		return nil, "", err
	}

	var checkedBy, approvedBy, checkedSigFile, approvedSigFile string
	if round, err := s.approvalRepo.GetLatestRound(ctx, prID); err == nil && round > 0 {
		if approvals, err := s.approvalRepo.ListByPRRound(ctx, prID, round); err == nil {
			lowestLevel, highestLevel := 0, 0
			for _, a := range approvals {
				if a.Status != "approved" {
					continue
				}
				if lowestLevel == 0 || a.Level < lowestLevel {
					lowestLevel = a.Level
					checkedBy = a.AdminName
					checkedSigFile = a.SignatureFile
				}
				if a.Level > highestLevel {
					highestLevel = a.Level
					approvedBy = a.AdminName
					approvedSigFile = a.SignatureFile
				}
			}
			if lowestLevel == highestLevel {
				approvedBy = ""
				approvedSigFile = ""
			}
		}
	}

	var paymentType, bank, bankAccNo, bankAccName, postedBy string
	if payments, err := s.paymentRepo.ListByPR(ctx, prID); err == nil && len(payments) > 0 {
		latest := payments[len(payments)-1]
		paymentType = latest.Type
		if latest.Bank != nil {
			bank = *latest.Bank
		}
		if latest.BankAccountNo != nil {
			bankAccNo = *latest.BankAccountNo
		}
		if latest.BankAccountName != nil {
			bankAccName = *latest.BankAccountName
		}
		if latest.Status == "paid" {
			postedBy = latest.AdminPaidName
		}
	}

	margin := pr.PoAmount - pr.Hpp
	marginPct := 0.0
	if pr.PoAmount != 0 {
		marginPct = margin / pr.PoAmount * 100
	}

	quotNo := ""
	if pr.QoutNo != nil {
		quotNo = *pr.QoutNo
	}

	poNoDisplay := "PO BELUM RELEASE"

	var assets []signatureAsset
	attach := func(relFile string) string {
		img := s.loadSignatureImage(relFile)
		if img == nil {
			return ""
		}
		idx := len(assets) + 1
		a := signatureAsset{
			relID:     fmt.Sprintf("rId%d", idx+3),
			fileName:  fmt.Sprintf("image%d.%s", idx, img.ext),
			data:      img.data,
			widthEMU:  img.widthEMU,
			heightEMU: img.heightEMU,
		}
		assets = append(assets, a)
		return rfpDrawingPara(a, 100+idx)
	}

	preparedSigXML := ""
	if sig, err := s.signatureRepo.GetLatestByAdmin(ctx, pr.RefAdmin); err == nil {
		preparedSigXML = attach(sig.File)
	}
	checkedSigXML := attach(checkedSigFile)
	approvedSigXML := attach(approvedSigFile)

	companyName := pr.ResponsibleName
	if companyName == "" {
		companyName = defaultCompanyName
	}

	fields := rfpFields{
		companyName: 			companyName,
		RfpNo:                 	pr.RfpNo,
		QuotNo:                 quotNo,
		PoNo:                   poNoDisplay,
		PaymentType:            paymentType,
		Bank:                   bank,
		BankAccNo:              bankAccNo,
		BankAccName:            bankAccName,
		Description:            pr.DescriptionItem,
		Amount:                 formatRupiah(pr.Hpp),
		PoAmount:               formatRupiah(pr.PoAmount),
		HppAmount:               formatRupiah(pr.Hpp),
		TotalAmount:             formatRupiah(pr.Hpp),
		MarginAmount:            formatRupiah(margin),
		MarginPercent:           fmt.Sprintf("%.1f%%", truncate1(marginPct)),
		TargetInvoice:           formatTargetInvoiceLine(pr.TargetInvoiceDate),
		PreparedBy:              pr.AdminName,
		CheckedBy:               checkedBy,
		ApprovedBy:              approvedBy,
		PostedBy:                postedBy,
		PreparedSignatureXML:    preparedSigXML,
		CheckedSignatureXML:     checkedSigXML,
		ApprovedSignatureXML:    approvedSigXML,
	}

	data, err := buildDocxZip(buildRFPDocumentXML(fields), assets)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("RFP_%s.docx", sanitizeFilename(pr.RfpNo))
	return data, filename, nil
}

// ---------------------------------------------------------------------
// Signature image handling
// ---------------------------------------------------------------------

// signatureAsset merepresentasikan satu gambar tanda tangan yang akan
// disisipkan ke dalam docx (media file + relationship + ukuran tampil).
type signatureAsset struct {
	relID     string // rIdN, dipakai di document.xml.rels & <a:blip r:embed=...>
	fileName  string // nama file relatif di dalam word/media/
	data      []byte
	widthEMU  int64
	heightEMU int64
}

type rawSignatureImage struct {
	data      []byte
	ext       string // "png" | "jpeg"
	widthEMU  int64
	heightEMU int64
}

// loadSignatureImage membaca file tanda tangan dari disk (path relatif yang
// tersimpan di DB, mis. "uploads/signature/xxx.png") dan menghitung ukuran
// tampil (EMU) dengan tinggi tetap 0.55in, lebar mengikuti rasio asli
// (dibatasi maksimal 1.6in supaya tidak melebar keluar sel tanda tangan).
// Mengembalikan nil (bukan error) untuk setiap kegagalan - tanda tangan
// adalah pelengkap tampilan, bukan bagian yang boleh menggagalkan export.
func (s *PRExportService) loadSignatureImage(relFile string) *rawSignatureImage {
	if relFile == "" || s.uploadDir == "" {
		return nil
	}
	fullPath := filepath.Join(s.uploadDir, "..", relFile)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	ext := "png"
	if format == "jpeg" {
		ext = "jpeg"
	}

	const targetHeightEMU = int64(0.55 * 914400)
	const maxWidthEMU = int64(1.6 * 914400)
	widthEMU, heightEMU := targetHeightEMU, targetHeightEMU
	if cfg.Height > 0 {
		ratio := float64(cfg.Width) / float64(cfg.Height)
		widthEMU = int64(float64(targetHeightEMU) * ratio)
		if widthEMU > maxWidthEMU {
			scale := float64(maxWidthEMU) / float64(widthEMU)
			widthEMU = maxWidthEMU
			heightEMU = int64(float64(heightEMU) * scale)
		}
	}
	return &rawSignatureImage{data: data, ext: ext, widthEMU: widthEMU, heightEMU: heightEMU}
}

// rfpDrawingPara membangun paragraf berisi gambar inline (WordprocessingML
// Drawing) yang mereferensikan relationship gambar lewat r:embed.
func rfpDrawingPara(a signatureAsset, docID int) string {
	return fmt.Sprintf(
		`<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:drawing>`+
			`<wp:inline distT="0" distB="0" distL="0" distR="0" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing">`+
			`<wp:extent cx="%d" cy="%d"/>`+
			`<wp:docPr id="%d" name="Signature%d"/>`+
			`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">`+
			`<a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
			`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
			`<pic:nvPicPr><pic:cNvPr id="%d" name="Signature%d"/><pic:cNvPicPr/></pic:nvPicPr>`+
			`<pic:blipFill><a:blip r:embed="%s" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`+
			`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>`+
			`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`,
		a.widthEMU, a.heightEMU, docID, docID, docID, docID, a.relID, a.widthEMU, a.heightEMU,
	)
}

// ---------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------

func formatTargetInvoiceLine(t *time.Time) string {
	if t == nil {
		return ""
	}
	return "TARGET INVOICE : " + formatIndonesianDate(*t)
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "_")
	s = replacer.Replace(s)
	if s == "" {
		return "PR"
	}
	return s
}

func formatIndonesianDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	days := [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months := [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli",
		"Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%s, %d %s %d", days[int(t.Weekday())], t.Day(), months[int(t.Month())], t.Year())
}

// formatRupiah -> "Rp. 880.000" (titik sebagai pemisah ribuan, tanpa desimal).
func formatRupiah(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	n := int64(math.Round(v))
	s := strconv.FormatInt(n, 10)
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, s[i])
	}
	res := "Rp. " + string(out)
	if neg {
		res = "-" + res
	}
	return res
}

// truncate1 memotong (bukan membulatkan) ke 1 desimal - konvensi yang
// dipakai form RFP manual supaya persentase margin tidak "digelembungkan"
// oleh pembulatan ke atas.
func truncate1(v float64) float64 {
	return math.Trunc(v*10) / 10
}

// ---------------------------------------------------------------------
// OOXML (docx) builder - murni string building, tanpa library eksternal
// ---------------------------------------------------------------------

type rfpFields struct {
	companyName string
	RfpNo         string
	QuotNo        string
	PoNo          string
	PaymentType   string
	Bank          string
	BankAccNo     string
	BankAccName   string
	Description   string
	Amount        string
	PoAmount      string
	HppAmount     string
	TotalAmount   string
	MarginAmount  string
	MarginPercent string
	TargetInvoice string // baris siap tampil, "" jika tidak ada target invoice date

	PreparedBy string
	CheckedBy  string
	ApprovedBy string
	PostedBy   string

	// Paragraf <w:drawing> siap pakai (hasil rfpDrawingPara), "" jika admin
	// terkait belum punya tanda tangan terdaftar / file gagal dibaca.
	PreparedSignatureXML string
	CheckedSignatureXML  string
	ApprovedSignatureXML string
}

const rfpPageWidth = 9026 // twips, area usable A4 dikurangi margin 1440 kiri-kanan

type paraOpt struct {
	Bold  bool
	Align string // "" (=left) | "center" | "right"
	Size  int    // half-points, default 20 (=10pt)
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func rfpPara(text string, o paraOpt) string {
	size := o.Size
	if size == 0 {
		size = 20
	}
	rpr := `<w:rFonts w:ascii="Arial" w:hAnsi="Arial"/>`
	if o.Bold {
		rpr += `<w:b/>`
	}
	rpr += fmt.Sprintf(`<w:sz w:val="%d"/>`, size)

	jc := ""
	if o.Align != "" && o.Align != "left" {
		jc = fmt.Sprintf(`<w:jc w:val="%s"/>`, o.Align)
	}
	return fmt.Sprintf(`<w:p><w:pPr>%s</w:pPr><w:r><w:rPr>%s</w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p>`,
		jc, rpr, xmlEscape(text))
}

func rfpEmptyPara() string { return `<w:p/>` }

func rfpCheckbox(selected bool, label string) string {
	box := "☐"
	if selected {
		box = "☑"
	}
	return box + " " + label
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

type cellOpt struct {
	Width int
	Span  int
	Shade string // hex fill, "" = tanpa shading
}

func rfpCell(content string, o cellOpt) string {
	span := ""
	if o.Span > 1 {
		span = fmt.Sprintf(`<w:gridSpan w:val="%d"/>`, o.Span)
	}
	shd := ""
	if o.Shade != "" {
		shd = fmt.Sprintf(`<w:shd w:val="clear" w:color="auto" w:fill="%s"/>`, o.Shade)
	}
	return fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s%s<w:vAlign w:val="center"/></w:tcPr>%s</w:tc>`,
		o.Width, span, shd, content)
}

func rfpRow(cells []string, heightTwips int) string {
	trPr := ""
	if heightTwips > 0 {
		trPr = fmt.Sprintf(`<w:trPr><w:trHeight w:val="%d"/></w:trPr>`, heightTwips)
	}
	return fmt.Sprintf(`<w:tr>%s%s</w:tr>`, trPr, strings.Join(cells, ""))
}

const rfpBorders = `<w:tblBorders>` +
	`<w:top w:val="single" w:sz="4" w:space="0" w:color="000000"/>` +
	`<w:left w:val="single" w:sz="4" w:space="0" w:color="000000"/>` +
	`<w:bottom w:val="single" w:sz="4" w:space="0" w:color="000000"/>` +
	`<w:right w:val="single" w:sz="4" w:space="0" w:color="000000"/>` +
	`<w:insideH w:val="single" w:sz="4" w:space="0" w:color="000000"/>` +
	`<w:insideV w:val="single" w:sz="4" w:space="0" w:color="000000"/>` +
	`</w:tblBorders>`

func rfpTable(gridWidths []int, rows []string) string {
	grid, total := "", 0
	for _, w := range gridWidths {
		grid += fmt.Sprintf(`<w:gridCol w:w="%d"/>`, w)
		total += w
	}
	tblPr := fmt.Sprintf(`<w:tblPr><w:tblW w:w="%d" w:type="dxa"/>%s<w:tblLayout w:type="fixed"/></w:tblPr>`, total, rfpBorders)
	return fmt.Sprintf(`<w:tbl>%s<w:tblGrid>%s</w:tblGrid>%s</w:tbl>`, tblPr, grid, strings.Join(rows, ""))
}

func buildRFPDocumentXML(f rfpFields) string {
	// --- Tabel 1: Judul ---
	titleContent := rfpPara("REQUEST FOR PAYMENT", paraOpt{Bold: true, Align: "center", Size: 28}) +
		rfpPara(f.companyName, paraOpt{Bold: true, Align: "center", Size: 28})
	table1 := rfpTable([]int{rfpPageWidth}, []string{
		rfpRow([]string{rfpCell(titleContent, cellOpt{Width: rfpPageWidth})}, 900),
	})

	// --- Tabel 2: Info RFP & Pembayaran ---
	w1, w2, w3, w4 := 1500, 2726, 2200, 2600
	paymentLine := rfpCheckbox(strings.EqualFold(f.PaymentType, "transfer"), "Transfer") +
		"     " + rfpCheckbox(strings.EqualFold(f.PaymentType, "cash"), "Cash")

	table2 := rfpTable([]int{w1, w2, w3, w4}, []string{
		rfpRow([]string{
			rfpCell(rfpPara("No. RFP", paraOpt{}), cellOpt{Width: w1}),
			rfpCell(rfpPara(f.RfpNo, paraOpt{}), cellOpt{Width: w2}),
			rfpCell(rfpPara("Payment Type", paraOpt{}), cellOpt{Width: w3}),
			rfpCell(rfpPara(paymentLine, paraOpt{}), cellOpt{Width: w4}),
		}, 0),
		rfpRow([]string{
			// rfpCell(rfpPara("Date", paraOpt{}), cellOpt{Width: w1}),
			// rfpCell(rfpPara(f.submitted_at, paraOpt{}), cellOpt{Width: w2}),
			rfpCell(rfpPara("Bank", paraOpt{}), cellOpt{Width: w1}),
			rfpCell(rfpPara(orDash(f.Bank), paraOpt{}), cellOpt{Width: w2}),
		}, 0),
		rfpRow([]string{
			rfpCell(rfpPara("No. Quot", paraOpt{}), cellOpt{Width: w1}),
			rfpCell(rfpPara(orDash(f.QuotNo), paraOpt{}), cellOpt{Width: w2}),
			rfpCell(rfpPara("Bank Account No.", paraOpt{}), cellOpt{Width: w3}),
			rfpCell(rfpPara(orDash(f.BankAccNo), paraOpt{}), cellOpt{Width: w4}),
		}, 0),
		rfpRow([]string{
			rfpCell(rfpPara("No. PO", paraOpt{}), cellOpt{Width: w1}),
			rfpCell(rfpPara(f.PoNo, paraOpt{}), cellOpt{Width: w2}),
			rfpCell(rfpPara("Bank Account Name", paraOpt{}), cellOpt{Width: w3}),
			rfpCell(rfpPara(orDash(f.BankAccName), paraOpt{}), cellOpt{Width: w4}),
		}, 0),
	})

	// --- Tabel 3: Description / Amount ---
	dw, aw := 6300, 2726
	rows3 := []string{
		rfpRow([]string{
			rfpCell(rfpPara("Description", paraOpt{Bold: true, Align: "center"}), cellOpt{Width: dw, Shade: "D9D9D9"}),
			rfpCell(rfpPara("Amount (Rp.)", paraOpt{Bold: true, Align: "center"}), cellOpt{Width: aw, Shade: "D9D9D9"}),
		}, 0),
		rfpRow([]string{
			rfpCell(rfpPara(f.Description, paraOpt{}), cellOpt{Width: dw}),
			rfpCell(rfpEmptyPara(), cellOpt{Width: aw}),
		}, 0),
		rfpRow([]string{
			rfpCell(rfpEmptyPara(), cellOpt{Width: dw}),
			rfpCell(rfpPara(f.Amount, paraOpt{Bold: true, Align: "right"}), cellOpt{Width: aw}),
		}, 0),
		rfpRow([]string{rfpCell(rfpPara("PO : "+f.PoAmount, paraOpt{}), cellOpt{Width: dw + aw, Span: 2})}, 0),
		rfpRow([]string{rfpCell(rfpPara("HPP : "+f.HppAmount+strings.Repeat(" ", 10)+"TOTAL : "+f.TotalAmount, paraOpt{}), cellOpt{Width: dw + aw, Span: 2})}, 0),
		rfpRow([]string{rfpCell(rfpPara("Margin : "+f.MarginAmount+" ("+f.MarginPercent+")", paraOpt{}), cellOpt{Width: dw + aw, Span: 2})}, 0),
	}
	if f.TargetInvoice != "" {
		rows3 = append(rows3, rfpRow([]string{rfpCell(rfpPara(f.TargetInvoice, paraOpt{Bold: true}), cellOpt{Width: dw + aw, Span: 2})}, 0))
	}
	rows3 = append(rows3,
		rfpRow([]string{rfpCell(rfpEmptyPara(), cellOpt{Width: dw + aw, Span: 2})}, 0),
		rfpRow([]string{
			rfpCell(rfpPara("Total Amount", paraOpt{Bold: true}), cellOpt{Width: dw, Shade: "D9D9D9"}),
			rfpCell(rfpPara(f.TotalAmount, paraOpt{Bold: true, Align: "right"}), cellOpt{Width: aw, Shade: "D9D9D9"}),
		}, 0),
	)
	table3 := rfpTable([]int{dw, aw}, rows3)

	// --- Tabel 4: Tanda tangan ---
	preparedCell := rfpEmptyPara()
	if f.PreparedSignatureXML != "" {
		preparedCell = f.PreparedSignatureXML
	}
	checkedCell := rfpEmptyPara()
	if f.CheckedSignatureXML != "" {
		checkedCell = f.CheckedSignatureXML
	}
	approvedCell := rfpEmptyPara()
	if f.ApprovedSignatureXML != "" {
		approvedCell = f.ApprovedSignatureXML
	}

	sw := rfpPageWidth / 4
	table4 := rfpTable([]int{sw, sw, sw, sw}, []string{
		rfpRow([]string{
			rfpCell(rfpPara("Prepared by :", paraOpt{Bold: true, Align: "center"}), cellOpt{Width: sw, Shade: "D9D9D9"}),
			rfpCell(rfpPara("Checked by :", paraOpt{Bold: true, Align: "center"}), cellOpt{Width: sw, Shade: "D9D9D9"}),
			rfpCell(rfpPara("Approved by :", paraOpt{Bold: true, Align: "center"}), cellOpt{Width: sw, Shade: "D9D9D9"}),
			rfpCell(rfpPara("Posted by :", paraOpt{Bold: true, Align: "center"}), cellOpt{Width: sw, Shade: "D9D9D9"}),
		}, 0),
		rfpRow([]string{
			rfpCell(preparedCell, cellOpt{Width: sw}),
			rfpCell(checkedCell, cellOpt{Width: sw}),
			rfpCell(approvedCell, cellOpt{Width: sw}),
			rfpCell(rfpEmptyPara(), cellOpt{Width: sw}), // Posted by: tidak ada data tanda tangan
		}, 900),
		rfpRow([]string{
			rfpCell(rfpPara(f.PreparedBy, paraOpt{Align: "center"}), cellOpt{Width: sw}),
			rfpCell(rfpPara(f.CheckedBy, paraOpt{Align: "center"}), cellOpt{Width: sw}),
			rfpCell(rfpPara(f.ApprovedBy, paraOpt{Align: "center"}), cellOpt{Width: sw}),
			rfpCell(rfpPara(f.PostedBy, paraOpt{Align: "center"}), cellOpt{Width: sw}),
		}, 0),
	})

	const sectPr = `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/>` +
		`<w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`

	body := table1 + rfpEmptyPara() + table2 + rfpEmptyPara() + table3 + rfpEmptyPara() + table4 + rfpEmptyPara() + sectPr

	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` + body + `</w:body></w:document>`
}

// ---------------------------------------------------------------------
// Static docx parts + zip packaging
// ---------------------------------------------------------------------

const rfpContentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Default Extension="jpeg" ContentType="image/jpeg"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>
  <Override PartName="/word/fontTable.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.fontTable+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`

const rfpRootRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`

// rfpDocumentRelsBase memuat relationship statis (styles/settings/fontTable,
// rId1-3). Relationship gambar tanda tangan (rId4, rId5, ...) disisipkan
// dinamis oleh buildDocxZip sesuai jumlah signatureAsset yang tersedia.
const rfpDocumentRelsBase = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/fontTable" Target="fontTable.xml"/>
</Relationships>`

const rfpStylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:docDefaults>
    <w:rPrDefault><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial" w:cs="Arial"/><w:sz w:val="20"/></w:rPr></w:rPrDefault>
  </w:docDefaults>
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
</w:styles>`

const rfpSettingsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:defaultTabStop w:val="708"/>
</w:settings>`

const rfpFontTableXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:fonts xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:font w:name="Arial"><w:family w:val="swiss"/></w:font>
</w:fonts>`

const rfpCoreXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <dc:title>Request For Payment</dc:title>
</cp:coreProperties>`

const rfpAppXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties">
  <Application>rms-backend</Application>
</Properties>`

func buildDocxZip(documentXML string, assets []signatureAsset) ([]byte, error) {
	rels := rfpDocumentRelsBase
	for _, a := range assets {
		entry := fmt.Sprintf(
			`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/%s"/>`,
			a.relID, a.fileName)
		rels = strings.Replace(rels, "</Relationships>", entry+"</Relationships>", 1)
	}

	type part struct {
		name    string
		content []byte
	}
	files := []part{
		{"[Content_Types].xml", []byte(rfpContentTypesXML)},
		{"_rels/.rels", []byte(rfpRootRelsXML)},
		{"word/document.xml", []byte(documentXML)},
		{"word/_rels/document.xml.rels", []byte(rels)},
		{"word/styles.xml", []byte(rfpStylesXML)},
		{"word/settings.xml", []byte(rfpSettingsXML)},
		{"word/fontTable.xml", []byte(rfpFontTableXML)},
		{"docProps/core.xml", []byte(rfpCoreXML)},
		{"docProps/app.xml", []byte(rfpAppXML)},
	}
	for _, a := range assets {
		files = append(files, part{"word/media/" + a.fileName, a.data})
	}

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(f.content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}