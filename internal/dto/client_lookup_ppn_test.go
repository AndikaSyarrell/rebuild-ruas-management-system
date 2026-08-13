package dto

import (
	"testing"

	"rms-backend/internal/utils"
)

// --- ClientRequest ---

func TestClientRequest_Normalize(t *testing.T) {
	req := ClientRequest{
		Name:    "  PT Contoh  ",
		Email:   "  Contoh@Example.COM  ",
		Phone:   "  0812  ",
		Address: "  Jl. Sudirman  ",
	}
	req.Normalize()
	if req.Name != "PT Contoh" {
		t.Errorf("expected trimmed name, got %q", req.Name)
	}
	if req.Email != "contoh@example.com" {
		t.Errorf("expected normalized email, got %q", req.Email)
	}
	if req.Phone != "0812" {
		t.Errorf("expected trimmed phone, got %q", req.Phone)
	}
	if req.Address != "Jl. Sudirman" {
		t.Errorf("expected trimmed address, got %q", req.Address)
	}
}

func TestClientRequest_Valid(t *testing.T) {
	req := ClientRequest{Name: "PT Contoh", Email: "contoh@example.com", RegionID: utils.FlexInt(1)}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestClientRequest_MissingName(t *testing.T) {
	req := ClientRequest{Email: "a@b.com", RegionID: utils.FlexInt(1)}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestClientRequest_MissingEmail(t *testing.T) {
	req := ClientRequest{Name: "PT Contoh", RegionID: utils.FlexInt(1)}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing email")
	}
}

func TestClientRequest_MissingRegion(t *testing.T) {
	req := ClientRequest{Name: "PT Contoh", Email: "a@b.com", RegionID: utils.FlexInt(0)}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing region_id")
	}
}

// --- TitleRequest (Region/Division/Unit) ---

func TestTitleRequest_Normalize(t *testing.T) {
	req := TitleRequest{Title: "  Jakarta  "}
	req.Normalize()
	if req.Title != "Jakarta" {
		t.Errorf("expected trimmed title, got %q", req.Title)
	}
}

func TestTitleRequest_Valid(t *testing.T) {
	req := TitleRequest{Title: "Jakarta"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestTitleRequest_Empty(t *testing.T) {
	req := TitleRequest{Title: ""}
	if err := req.Validate(); err == nil {
		t.Error("expected error for empty title")
	}
}

func TestTitleRequest_WhitespaceOnlyFailsAfterNormalize(t *testing.T) {
	// Judul yang cuma spasi harus dianggap kosong SETELAH Normalize()
	// dipanggil (pola pemakaian nyata: Normalize() lalu Validate()).
	req := TitleRequest{Title: "   "}
	req.Normalize()
	if err := req.Validate(); err == nil {
		t.Error("expected whitespace-only title to fail validation after normalize")
	}
}

// --- PpnRequest ---

func TestPpnRequest_ValidRange(t *testing.T) {
	for _, v := range []float64{0, 11, 12, 100} {
		req := PpnRequest{Value: utils.FlexFloat(v)}
		if err := req.Validate(); err != nil {
			t.Errorf("expected value=%v to be valid, got error: %v", v, err)
		}
	}
}

func TestPpnRequest_NegativeRejected(t *testing.T) {
	req := PpnRequest{Value: utils.FlexFloat(-1)}
	if err := req.Validate(); err == nil {
		t.Error("expected error for negative PPN value")
	}
}

func TestPpnRequest_OverHundredRejected(t *testing.T) {
	req := PpnRequest{Value: utils.FlexFloat(100.01)}
	if err := req.Validate(); err == nil {
		t.Error("expected error for PPN value over 100")
	}
}
