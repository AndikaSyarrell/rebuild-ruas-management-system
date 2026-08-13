package utils

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePagination_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/po", nil)
	p := ParsePagination(r)
	if p.Page != 1 {
		t.Errorf("expected default page=1, got %d", p.Page)
	}
	if p.PerPage != 10 {
		t.Errorf("expected default per_page=10, got %d", p.PerPage)
	}
	if p.Offset != 0 {
		t.Errorf("expected offset=0, got %d", p.Offset)
	}
}

func TestParsePagination_Custom(t *testing.T) {
	r := httptest.NewRequest("GET", "/po?page=3&item=20", nil)
	p := ParsePagination(r)
	if p.Page != 3 {
		t.Errorf("expected page=3, got %d", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("expected per_page=20, got %d", p.PerPage)
	}
	if p.Offset != 40 {
		t.Errorf("expected offset=40, got %d", p.Offset)
	}
}

func TestParsePagination_InvalidPageFallsBackToDefault(t *testing.T) {
	r := httptest.NewRequest("GET", "/po?page=-5&item=abc", nil)
	p := ParsePagination(r)
	if p.Page != 1 {
		t.Errorf("expected negative page to fall back to 1, got %d", p.Page)
	}
	if p.PerPage != 10 {
		t.Errorf("expected non-numeric item to fall back to 10, got %d", p.PerPage)
	}
}

func TestParsePagination_CapsPerPageAt200(t *testing.T) {
	// Mencegah client meminta seluruh tabel sekaligus.
	r := httptest.NewRequest("GET", "/po?item=999999", nil)
	p := ParsePagination(r)
	if p.PerPage != 200 {
		t.Errorf("expected per_page capped at 200, got %d", p.PerPage)
	}
}

func TestTotalPage_ExactDivision(t *testing.T) {
	if got := TotalPage(20, 10); got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
}

func TestTotalPage_RoundsUp(t *testing.T) {
	if got := TotalPage(21, 10); got != 3 {
		t.Errorf("expected 3 (rounded up), got %d", got)
	}
}

func TestTotalPage_ZeroData(t *testing.T) {
	if got := TotalPage(0, 10); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestTotalPage_ZeroPerPage(t *testing.T) {
	// Menghindari division by zero.
	if got := TotalPage(100, 0); got != 0 {
		t.Errorf("expected 0 when per_page is 0, got %d", got)
	}
}

func TestParseDateParam_ISOFormat(t *testing.T) {
	got := ParseDateParam("2026-07-16")
	if got != "2026-07-16" {
		t.Errorf("expected 2026-07-16, got %s", got)
	}
}

func TestParseDateParam_EmptyInput(t *testing.T) {
	if got := ParseDateParam(""); got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

func TestParseDateParam_InvalidFormat_FailsClosed(t *testing.T) {
	// Format tak dikenali harus mengembalikan string kosong (fail-closed),
	// BUKAN meloloskan input mentah apa adanya ke query SQL seperti PHP asli.
	got := ParseDateParam("not-a-date-at-all")
	if got != "" {
		t.Errorf("expected empty string for unparseable date, got %s", got)
	}
}

func TestParseDateParam_RFC3339(t *testing.T) {
	got := ParseDateParam("2026-07-16T10:30:00Z")
	if got != "2026-07-16" {
		t.Errorf("expected normalized date 2026-07-16, got %s", got)
	}
}

func TestRandomHex_CorrectLength(t *testing.T) {
	h := RandomHex(20)
	// n byte -> 2n hex character.
	if len(h) != 40 {
		t.Errorf("expected 40 hex chars for 20 bytes, got %d", len(h))
	}
}

func TestRandomHex_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		h := RandomHex(16)
		if seen[h] {
			t.Fatalf("collision detected at iteration %d: %s", i, h)
		}
		seen[h] = true
	}
}

func TestGenerateSequentialID_HasPrefix(t *testing.T) {
	id := GenerateSequentialID("PO")
	if !strings.HasPrefix(id, "PO") {
		t.Errorf("expected ID to start with PO, got %s", id)
	}
}

func TestGenerateSequentialID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := GenerateSequentialID("ADM")
		if seen[id] {
			t.Fatalf("collision detected: %s", id)
		}
		seen[id] = true
	}
}

func TestAtoiDefault_ValidInput(t *testing.T) {
	if got := AtoiDefault("42", 0); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestAtoiDefault_InvalidInputUsesDefault(t *testing.T) {
	if got := AtoiDefault("not-a-number", 99); got != 99 {
		t.Errorf("expected fallback default 99, got %d", got)
	}
}

func TestAtoiDefault_EmptyInputUsesDefault(t *testing.T) {
	if got := AtoiDefault("", 5); got != 5 {
		t.Errorf("expected fallback default 5, got %d", got)
	}
}

func TestParseIDParam_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	id, ok := ParseIDParam(w, "123")
	if !ok {
		t.Fatal("expected ok=true for valid numeric ID")
	}
	if id != 123 {
		t.Errorf("expected 123, got %d", id)
	}
	if w.Code != 200 {
		t.Errorf("expected no error response written, got status %d", w.Code)
	}
}

func TestParseIDParam_Invalid_Writes400(t *testing.T) {
	// Regression guard untuk bug lama: id, _ := strconv.Atoi(...) yang diam-diam
	// menjadikan ID tidak valid sebagai 0. ParseIDParam WAJIB menolak eksplisit.
	w := httptest.NewRecorder()
	id, ok := ParseIDParam(w, "abc")
	if ok {
		t.Fatal("expected ok=false for non-numeric ID")
	}
	if id != 0 {
		t.Errorf("expected zero-value id on failure, got %d", id)
	}
	if w.Code != 400 {
		t.Errorf("expected HTTP 400 response written, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ID tidak valid") {
		t.Errorf("expected error message about invalid ID, got body: %s", w.Body.String())
	}
}

func TestParseIDParam_EmptyString(t *testing.T) {
	w := httptest.NewRecorder()
	_, ok := ParseIDParam(w, "")
	if ok {
		t.Error("expected ok=false for empty string")
	}
}
