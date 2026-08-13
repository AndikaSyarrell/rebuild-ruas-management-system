package utils

import (
	"encoding/json"
	"testing"
)

func TestFlexInt_UnmarshalJSON_Number(t *testing.T) {
	var f FlexInt
	if err := json.Unmarshal([]byte(`5`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int(f) != 5 {
		t.Errorf("expected 5, got %d", int(f))
	}
}

func TestFlexInt_UnmarshalJSON_String(t *testing.T) {
	var f FlexInt
	if err := json.Unmarshal([]byte(`"5"`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int(f) != 5 {
		t.Errorf("expected 5, got %d", int(f))
	}
}

func TestFlexInt_UnmarshalJSON_EmptyString(t *testing.T) {
	var f FlexInt
	if err := json.Unmarshal([]byte(`""`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int(f) != 0 {
		t.Errorf("expected 0 for empty string, got %d", int(f))
	}
}

func TestFlexInt_UnmarshalJSON_Null(t *testing.T) {
	var f FlexInt
	if err := json.Unmarshal([]byte(`null`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int(f) != 0 {
		t.Errorf("expected 0 for null, got %d", int(f))
	}
}

func TestFlexInt_UnmarshalJSON_InvalidString(t *testing.T) {
	var f FlexInt
	if err := json.Unmarshal([]byte(`"abc"`), &f); err == nil {
		t.Error("expected error for non-numeric string, got nil")
	}
}

func TestFlexInt_UnmarshalJSON_InvalidType(t *testing.T) {
	var f FlexInt
	if err := json.Unmarshal([]byte(`true`), &f); err == nil {
		t.Error("expected error for boolean value, got nil")
	}
}

func TestFlexInt_MarshalJSON(t *testing.T) {
	f := FlexInt(42)
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "42" {
		t.Errorf("expected raw number 42, got %s", string(b))
	}
}

func TestFlexInt_TypeCast(t *testing.T) {
	// Konfirmasi FlexInt bisa langsung di-cast ke int (underlying type),
	// pola yang dipakai di seluruh handler tanpa method wrapper.
	f := FlexInt(7)
	var i int = int(f)
	if i != 7 {
		t.Errorf("expected 7, got %d", i)
	}
}

func TestFlexInt_InStruct(t *testing.T) {
	type payload struct {
		Qty    FlexInt `json:"qty"`
		UnitID FlexInt `json:"unit_id"`
	}
	var p payload
	raw := `{"qty": "10", "unit_id": 3}`
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int(p.Qty) != 10 {
		t.Errorf("expected qty=10, got %d", int(p.Qty))
	}
	if int(p.UnitID) != 3 {
		t.Errorf("expected unit_id=3, got %d", int(p.UnitID))
	}
}

func TestFlexInt_PointerField_Nil(t *testing.T) {
	type payload struct {
		DivisionID *FlexInt `json:"division_id"`
	}
	var p payload
	if err := json.Unmarshal([]byte(`{}`), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.DivisionID != nil {
		t.Error("expected DivisionID to remain nil when field is absent")
	}
}

func TestFlexInt_PointerField_Set(t *testing.T) {
	type payload struct {
		DivisionID *FlexInt `json:"division_id"`
	}
	var p payload
	if err := json.Unmarshal([]byte(`{"division_id": "2"}`), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.DivisionID == nil {
		t.Fatal("expected DivisionID to be set")
	}
	if int(*p.DivisionID) != 2 {
		t.Errorf("expected 2, got %d", int(*p.DivisionID))
	}
}

func TestFlexFloat_UnmarshalJSON_Number(t *testing.T) {
	var f FlexFloat
	if err := json.Unmarshal([]byte(`50000.5`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if float64(f) != 50000.5 {
		t.Errorf("expected 50000.5, got %f", float64(f))
	}
}

func TestFlexFloat_UnmarshalJSON_String(t *testing.T) {
	var f FlexFloat
	if err := json.Unmarshal([]byte(`"50000.5"`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if float64(f) != 50000.5 {
		t.Errorf("expected 50000.5, got %f", float64(f))
	}
}

func TestFlexFloat_UnmarshalJSON_EmptyString(t *testing.T) {
	var f FlexFloat
	if err := json.Unmarshal([]byte(`""`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if float64(f) != 0 {
		t.Errorf("expected 0, got %f", float64(f))
	}
}

func TestFlexFloat_UnmarshalJSON_InvalidString(t *testing.T) {
	var f FlexFloat
	if err := json.Unmarshal([]byte(`"not-a-number"`), &f); err == nil {
		t.Error("expected error for non-numeric string, got nil")
	}
}

func TestFlexFloat_MarshalJSON(t *testing.T) {
	f := FlexFloat(12.5)
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "12.5" {
		t.Errorf("expected 12.5, got %s", string(b))
	}
}

func TestFlexFloat_ZeroValue(t *testing.T) {
	// Regression guard: memastikan negative/zero tetap ter-decode dengan benar
	// (bukan cuma truthy check yang salah kaprah menganggap 0 sebagai "kosong").
	var f FlexFloat
	if err := json.Unmarshal([]byte(`0`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if float64(f) != 0 {
		t.Errorf("expected 0, got %f", float64(f))
	}
}
