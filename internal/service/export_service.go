package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type ExportConfig struct {
	// CacheDir adalah folder tempat file .xlsx hasil generate disimpan & dipakai
	// sebagai cache (nama file = hash filter, TTL dicek dari mtime file).
	CacheDir string
	// TemplatePath opsional. Jika diisi dan file-nya ada, baris header (baris 1)
	// di template tsb dicocokkan (case-insensitive) terhadap Label kolom bawaan
	// (lihat exportColumns()) untuk menentukan kolom mana & urutan mana yang
	// dipakai - sehingga template bisa disusun ulang tanpa ubah kode. Kosongkan
	// untuk selalu men-generate sheet baru dengan seluruh kolom bawaan.
	TemplatePath string
	// TTL masa berlaku cache sebelum file di-generate ulang.
	TTL time.Duration
}

type ExportService struct {
	repo  *repository.PORepo
	cfg   ExportConfig
	group utils.SingleflightGroup
}

func NewExportService(repo *repository.PORepo, cfg ExportConfig) *ExportService {
	_ = os.MkdirAll(cfg.CacheDir, 0o755)
	return &ExportService{repo: repo, cfg: cfg}
}

type exportColumn struct {
	Label   string
	Value   func(r repository.ExportRow) interface{}
	Numeric bool
}

// exportColumns adalah daftar kolom bawaan (dipakai apa adanya jika tidak ada
// template, atau sebagai kamus pencocokan label jika ada template).
func exportColumns() []exportColumn {
	return []exportColumn{
		{"No PO", func(r repository.ExportRow) interface{} { return r.OrderNum }, false},
		{"Invoice", func(r repository.ExportRow) interface{} { return r.Invoice }, false},
		{"Status", func(r repository.ExportRow) interface{} { return r.Status }, false},
		{"Tanggal PO", func(r repository.ExportRow) interface{} { return r.Date }, false},
		{"Region", func(r repository.ExportRow) interface{} { return r.RegionTitle }, false},
		{"Divisi", func(r repository.ExportRow) interface{} { return r.DivisionName }, false},
		{"PIC", func(r repository.ExportRow) interface{} { return r.PicName }, false},
		{"Dibuat Oleh", func(r repository.ExportRow) interface{} { return r.AdminName }, false},
		{"Klien", func(r repository.ExportRow) interface{} { return r.ClientName }, false},
		{"Email Klien", func(r repository.ExportRow) interface{} { return r.ClientEmail }, false},
		{"Telepon Klien", func(r repository.ExportRow) interface{} { return r.ClientPhone }, false},
		{"Sub Client", func(r repository.ExportRow) interface{} { return r.SubClient }, false},
		{"Produk", func(r repository.ExportRow) interface{} { return r.ItemProduct }, false},
		{"Deskripsi", func(r repository.ExportRow) interface{} { return r.ItemDesc }, false},
		{"Satuan", func(r repository.ExportRow) interface{} { return r.UnitTitle }, false},
		{"Qty", func(r repository.ExportRow) interface{} { return r.ItemQty }, false},
		{"Harga Satuan", func(r repository.ExportRow) interface{} { return r.ItemPrice }, true},
		{"Total Item", func(r repository.ExportRow) interface{} { return r.ItemPrice * float64(r.ItemQty) }, true},
		{"Subtotal PO", func(r repository.ExportRow) interface{} { return r.Subtotal }, true},
		{"PPN (%)", func(r repository.ExportRow) interface{} { return r.PpnRate }, false},
		{"Jumlah PPN", func(r repository.ExportRow) interface{} { return r.PpnAmount }, true},
		{"Total PO", func(r repository.ExportRow) interface{} { return r.Total }, true},
		{"Status Bayar", func(r repository.ExportRow) interface{} { return r.Paid }, false},
		{"Catatan", func(r repository.ExportRow) interface{} { return r.Notes }, false},
	}
}

func exportCacheKey(f repository.ExportFilter) string {
	raw := fmt.Sprintf("start=%s&end=%s&division=%d&status=%s&region=%d&client=%d",
		f.StartDate, f.EndDate, f.DivisionID, f.Status, f.RegionID, f.ClientID)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// GetOrGenerate mengembalikan path file .xlsx yang sudah jadi untuk filter
// tsb. Kalau ada file cache yang masih berlaku (umur < TTL), langsung dipakai
// tanpa query ke database sama sekali. Kalau tidak, file di-generate ulang -
// dan permintaan lain dengan filter identik yang datang BERSAMAAN akan
// menunggu hasil yang sama alih-alih ikut men-generate (lihat SingleflightGroup).
func (s *ExportService) GetOrGenerate(ctx context.Context, f repository.ExportFilter) (path string, cacheHit bool, err error) {
	key := exportCacheKey(f)
	path = filepath.Join(s.cfg.CacheDir, key+".xlsx")

	if s.isFresh(path) {
		return path, true, nil
	}

	v, err := s.group.Do(key, func() (interface{}, error) {
    if s.isFresh(path) {
        return path, nil
    }

    rows, err := s.repo.ExportItemRows(ctx, f)
    if err != nil {
        return nil, fmt.Errorf("query export: %w", err)
    }

    // Keep the .xlsx extension so excelize's SaveAs can detect the format;
    // put the random suffix in the filename stem instead.
    ext := filepath.Ext(path)                                  // ".xlsx"
    base := strings.TrimSuffix(path, ext)                      // ".../a1b2c3..."
    tmpPath := base + ".tmp-" + utils.RandomHex(6) + ext        // ".../a1b2c3....tmp-9f3e21.xlsx"

    if err := s.writeXlsx(tmpPath, rows); err != nil {
        os.Remove(tmpPath)
        return nil, fmt.Errorf("generate xlsx: %w", err)
    }
    if err := os.Rename(tmpPath, path); err != nil {
        os.Remove(tmpPath)
        return nil, fmt.Errorf("simpan file export: %w", err)
    }
    return path, nil
})
	if err != nil {
		return "", false, err
	}
	return v.(string), false, nil
}

func (s *ExportService) isFresh(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < s.cfg.TTL
}

func (s *ExportService) writeXlsx(path string, rows []repository.ExportRow) error {
	cols := exportColumns()

	f, dataStartRow, colOrder, usingTemplate, err := s.openSheet(cols)
	if err != nil {
		return fmt.Errorf("openSheet: %w", err)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)

	numStyle, err := f.NewStyle(&excelize.Style{NumFmt: 3})
	if err != nil {
		return fmt.Errorf("NewStyle numStyle: %w", err)
	}

	for i, row := range rows {
		excelRow := dataStartRow + i
		for colIdx, col := range colOrder {
			cellRef, err := excelize.CoordinatesToCellName(colIdx+1, excelRow)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cellRef, col.Value(row)); err != nil {
				return err
			}
			if col.Numeric {
				_ = f.SetCellStyle(sheet, cellRef, cellRef, numStyle)
			}
		}
	}

	// Kalau tidak pakai template, atur lebar kolom seadanya biar rapi dibaca.
	if !usingTemplate {
		for i, col := range colOrder {
			colLetter, err := excelize.ColumnNumberToName(i + 1)
			if err != nil {
				continue
			}
			width := float64(len(col.Label)) + 8
			if width < 12 {
				width = 12
			}
			_ = f.SetColWidth(sheet, colLetter, colLetter, width)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("MkdirAll: %w", err)
	}
	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("SaveAs: %w", err)
	}
	return nil
}

// openSheet menyiapkan workbook kerja: pakai template kalau tersedia & valid,
// atau bikin sheet baru dari nol dengan header bawaan.
func (s *ExportService) openSheet(cols []exportColumn) (f *excelize.File, dataStartRow int, colOrder []exportColumn, usingTemplate bool, err error) {
	if s.cfg.TemplatePath != "" {
		if _, statErr := os.Stat(s.cfg.TemplatePath); statErr == nil {
			tf, openErr := excelize.OpenFile(s.cfg.TemplatePath)
			if openErr == nil {
				sheet := tf.GetSheetName(0)
				allRows, rowsErr := tf.GetRows(sheet)
				if rowsErr == nil && len(allRows) > 0 {
					matched := matchTemplateColumns(allRows[0], cols)
					if len(matched) > 0 {
						return tf, 2, matched, true, nil
					}
				}
				tf.Close()
			}
		}
	}

	nf := excelize.NewFile()
	sheet := nf.GetSheetName(0)
	boldStyle, styleErr := nf.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if styleErr != nil {
		return nil, 0, nil, false, styleErr
	}
	for i, col := range cols {
		cellRef, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = nf.SetCellValue(sheet, cellRef, col.Label)
		_ = nf.SetCellStyle(sheet, cellRef, cellRef, boldStyle)
	}
	_ = nf.SetPanes(sheet, &excelize.Panes{
		Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft",
	})
	return nf, 2, cols, false, nil
}

// matchTemplateColumns mencocokkan baris header template (case-insensitive,
// trim spasi) terhadap Label kolom bawaan, sehingga urutan & subset kolom di
// template DIHORMATI - kolom yang tidak dikenali di template diabaikan.
func matchTemplateColumns(headerRow []string, cols []exportColumn) []exportColumn {
	var ordered []exportColumn
	for _, h := range headerRow {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		for _, c := range cols {
			if strings.EqualFold(h, c.Label) {
				ordered = append(ordered, c)
				break
			}
		}
	}
	return ordered
}
