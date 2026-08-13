package dto

import (
	"testing"

	"rms-backend/internal/utils"
)

// --- RoleRequest ---

func TestRoleRequest_Normalize(t *testing.T) {
	req := RoleRequest{Title: "  Warehouse Staff  "}
	req.Normalize()
	if req.Title != "Warehouse Staff" {
		t.Errorf("expected trimmed title, got %q", req.Title)
	}
}

func TestRoleRequest_Valid(t *testing.T) {
	req := RoleRequest{Title: "Warehouse Staff", AccessIDs: []utils.FlexInt{1, 2, 3}}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestRoleRequest_MissingTitle(t *testing.T) {
	req := RoleRequest{}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing title")
	}
}

func TestRoleRequest_EmptyAccessIDsAllowed(t *testing.T) {
	// Role tanpa akses sama sekali harus tetap valid secara struktural (role
	// baru yang belum di-assign akses apapun) - itu keputusan bisnis terpisah,
	// bukan pelanggaran validasi.
	req := RoleRequest{Title: "Empty Role"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected role with no access_ids to be valid, got error: %v", err)
	}
}

func TestRoleRequest_AccessIDInts_Conversion(t *testing.T) {
	req := RoleRequest{Title: "R", AccessIDs: []utils.FlexInt{1, 2, 3}}
	ints := req.AccessIDInts()
	if len(ints) != 3 {
		t.Fatalf("expected 3 ints, got %d", len(ints))
	}
	if ints[0] != 1 || ints[1] != 2 || ints[2] != 3 {
		t.Errorf("expected [1,2,3], got %v", ints)
	}
}

func TestRoleRequest_AccessIDInts_EmptySlice(t *testing.T) {
	req := RoleRequest{Title: "R"}
	ints := req.AccessIDInts()
	if len(ints) != 0 {
		t.Errorf("expected empty slice, got %v", ints)
	}
}

// --- AccessRequest ---

func TestAccessRequest_Normalize(t *testing.T) {
	req := AccessRequest{Title: "  Export PO  ", Module: "  po  ", Slug: "  Export_PO  "}
	req.Normalize()
	if req.Title != "Export PO" {
		t.Errorf("expected trimmed title, got %q", req.Title)
	}
	if req.Module != "po" {
		t.Errorf("expected trimmed module, got %q", req.Module)
	}
	if req.Slug != "export_po" {
		t.Errorf("expected trimmed+lowercased slug, got %q", req.Slug)
	}
}

func TestAccessRequest_Valid(t *testing.T) {
	req := AccessRequest{Title: "Export PO", Module: "po", Slug: "export_po"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestAccessRequest_MissingTitle(t *testing.T) {
	req := AccessRequest{Module: "po", Slug: "export_po"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing title")
	}
}

func TestAccessRequest_MissingModule(t *testing.T) {
	req := AccessRequest{Title: "Export PO", Slug: "export_po"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing module")
	}
}

func TestAccessRequest_MissingSlug(t *testing.T) {
	req := AccessRequest{Title: "Export PO", Module: "po"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing slug")
	}
}
