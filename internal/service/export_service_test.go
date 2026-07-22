package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"rms-backend/internal/repository"
)

func sampleRows() []repository.ExportRow {
	return []repository.ExportRow{
		{
			OrderNum: "PO-DEMO-0001", Status: "open", Date: "2026-07-16",
			RegionTitle: "Jakarta", DivisionName: "Sales", PicName: "Budi",
			AdminName: "Super Admin", ClientName: "PT Contoh", ClientEmail: "client@contoh.com",
			ClientPhone: "0812", SubClient: "", ItemProduct: "Kertas A4", ItemDesc: "80gsm",
			UnitTitle: "Pcs", ItemQty: 10, ItemPrice: 50000,
			Subtotal: 800000, PpnRate: 11, PpnAmount: 88000, Total: 888000, Paid: "no",
		},
		{
			OrderNum: "PO-DEMO-0001", Status: "open", Date: "2026-07-16",
			RegionTitle: "Jakarta", DivisionName: "Sales", PicName: "Budi",
			AdminName: "Super Admin", ClientName: "PT Contoh", ClientEmail: "client@contoh.com",
			ClientPhone: "0812", SubClient: "", ItemProduct: "Tinta Printer", ItemDesc: "Hitam",
			UnitTitle: "Pcs", ItemQty: 2, ItemPrice: 150000,
			Subtotal: 800000, PpnRate: 11, PpnAmount: 88000, Total: 888000, Paid: "no",
		},
	}
}

func TestWriteXlsxWithoutTemplate(t *testing.T) {
	dir := t.TempDir()
	svc := &ExportService{cfg: ExportConfig{CacheDir: dir, TTL: 20 * time.Minute}}

	path := filepath.Join(dir, "out.xlsx")
	if err := svc.writeXlsx(path, sampleRows()); err != nil {
		t.Fatalf("writeXlsx failed: %v", err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("cannot reopen generated xlsx: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatalf("GetRows failed: %v", err)
	}

	if len(rows) != 3 { // 1 header + 2 item rows
		t.Fatalf("expected 3 rows (1 header + 2 data), got %d", len(rows))
	}
	if rows[0][0] != "No PO" {
		t.Errorf("expected first header cell 'No PO', got %q", rows[0][0])
	}
	if rows[1][0] != "PO-DEMO-0001" {
		t.Errorf("expected first data cell 'PO-DEMO-0001', got %q", rows[1][0])
	}
	if rows[2][12] != "Tinta Printer" { // kolom "Produk" index ke-12 (0-based) sesuai exportColumns()
		t.Errorf("expected second item row product 'Tinta Printer', got %q", rows[2][12])
	}
}

func TestWriteXlsxWithTemplate_SubsetAndReorder(t *testing.T) {
	dir := t.TempDir()

	// Bikin template minimal: header cuma 3 kolom, sengaja dibalik urutannya
	// dan tidak semua kolom dipakai - membuktikan template MENENTUKAN subset & urutan.
	tmpl := excelize.NewFile()
	sheet := tmpl.GetSheetName(0)
	tmpl.SetCellValue(sheet, "A1", "Produk")
	tmpl.SetCellValue(sheet, "B1", "No PO")
	tmpl.SetCellValue(sheet, "C1", "Qty")
	templatePath := filepath.Join(dir, "template.xlsx")
	if err := tmpl.SaveAs(templatePath); err != nil {
		t.Fatalf("failed to save template fixture: %v", err)
	}
	tmpl.Close()

	svc := &ExportService{cfg: ExportConfig{CacheDir: dir, TemplatePath: templatePath, TTL: 20 * time.Minute}}
	outPath := filepath.Join(dir, "out.xlsx")
	if err := svc.writeXlsx(outPath, sampleRows()); err != nil {
		t.Fatalf("writeXlsx with template failed: %v", err)
	}

	f, err := excelize.OpenFile(outPath)
	if err != nil {
		t.Fatalf("cannot reopen generated xlsx: %v", err)
	}
	defer f.Close()
	sheetOut := f.GetSheetName(0)
	rows, err := f.GetRows(sheetOut)
	if err != nil {
		t.Fatalf("GetRows failed: %v", err)
	}

	if len(rows) < 2 {
		t.Fatalf("expected at least 2 rows, got %d", len(rows))
	}
	// Baris 1 harus tetap header template asli (tidak ditimpa)
	if rows[0][0] != "Produk" || rows[0][1] != "No PO" || rows[0][2] != "Qty" {
		t.Fatalf("template header should be preserved as-is, got %v", rows[0])
	}
	// Baris data pertama mengikuti urutan template: Produk, No PO, Qty
	if rows[1][0] != "Kertas A4" {
		t.Errorf("expected col A row2 = 'Kertas A4' (Produk), got %q", rows[1][0])
	}
	if rows[1][1] != "PO-DEMO-0001" {
		t.Errorf("expected col B row2 = 'PO-DEMO-0001' (No PO), got %q", rows[1][1])
	}
	if rows[1][2] != "10" {
		t.Errorf("expected col C row2 = '10' (Qty), got %q", rows[1][2])
	}
}

func TestExportCacheKeyDeterministic(t *testing.T) {
	f1 := repository.ExportFilter{StartDate: "2026-01-01", EndDate: "2026-12-31", RegionID: 1, Status: "open"}
	f2 := repository.ExportFilter{StartDate: "2026-01-01", EndDate: "2026-12-31", RegionID: 1, Status: "open"}
	f3 := repository.ExportFilter{StartDate: "2026-01-01", EndDate: "2026-12-31", RegionID: 2, Status: "open"}

	if exportCacheKey(f1) != exportCacheKey(f2) {
		t.Error("identical filters should produce identical cache keys")
	}
	if exportCacheKey(f1) == exportCacheKey(f3) {
		t.Error("different filters (region) should produce different cache keys")
	}
}

func TestIsFreshRespectsTTL(t *testing.T) {
	dir := t.TempDir()
	svc := &ExportService{cfg: ExportConfig{CacheDir: dir, TTL: 20 * time.Minute}}
	path := filepath.Join(dir, "cached.xlsx")

	if svc.isFresh(path) {
		t.Error("nonexistent file should never be considered fresh")
	}

	if err := os.WriteFile(path, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("failed to write dummy cache file: %v", err)
	}
	if !svc.isFresh(path) {
		t.Error("freshly written file should be considered fresh (within TTL)")
	}

	old := time.Now().Add(-30 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("failed to backdate file mtime: %v", err)
	}
	if svc.isFresh(path) {
		t.Error("file older than TTL should NOT be considered fresh")
	}
}
