package dto

import (
	"testing"

	"rms-backend/internal/utils"
)

func TestCreateAdminRequest_Normalize_TrimsAndLowercasesEmail(t *testing.T) {
	req := CreateAdminRequest{
		Email: "  NewAdmin@RMS.Local  ",
		Name:  "  New Admin  ",
	}
	req.Normalize()
	if req.Email != "newadmin@rms.local" {
		t.Errorf("expected normalized email, got %q", req.Email)
	}
	if req.Name != "New Admin" {
		t.Errorf("expected trimmed name, got %q", req.Name)
	}
}

func TestCreateAdminRequest_Normalize_DefaultsPicToNo(t *testing.T) {
	req := CreateAdminRequest{Email: "a@b.com", Name: "A"}
	req.Normalize()
	if req.Pic != "no" {
		t.Errorf("expected pic to default to 'no', got %q", req.Pic)
	}
	if req.PicClient != "no" {
		t.Errorf("expected pic_client to default to 'no', got %q", req.PicClient)
	}
}

func TestCreateAdminRequest_Valid(t *testing.T) {
	req := CreateAdminRequest{
		Email: "admin@rms.local", Name: "Admin", RegionID: utils.FlexInt(1),
		Pic: "yes", PicClient: "no",
	}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestCreateAdminRequest_MissingEmail(t *testing.T) {
	req := CreateAdminRequest{Name: "Admin", RegionID: utils.FlexInt(1)}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing email")
	}
}

func TestCreateAdminRequest_MissingName(t *testing.T) {
	req := CreateAdminRequest{Email: "a@b.com", RegionID: utils.FlexInt(1)}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestCreateAdminRequest_MissingRegion(t *testing.T) {
	req := CreateAdminRequest{Email: "a@b.com", Name: "Admin", RegionID: utils.FlexInt(0)}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing region_id")
	}
}

func TestCreateAdminRequest_InvalidPicValue(t *testing.T) {
	req := CreateAdminRequest{
		Email: "a@b.com", Name: "Admin", RegionID: utils.FlexInt(1), Pic: "maybe",
	}
	if err := req.Validate(); err == nil {
		t.Error("expected error for invalid pic value")
	}
}

func TestCreateAdminRequest_InvalidPicClientValue(t *testing.T) {
	req := CreateAdminRequest{
		Email: "a@b.com", Name: "Admin", RegionID: utils.FlexInt(1), Pic: "yes", PicClient: "sure",
	}
	if err := req.Validate(); err == nil {
		t.Error("expected error for invalid pic_client value")
	}
}

func TestCreateAdminRequest_RoleIDPtr_Nil(t *testing.T) {
	req := CreateAdminRequest{}
	if req.RoleIDPtr() != nil {
		t.Error("expected nil RoleIDPtr when RoleID not set")
	}
}

func TestCreateAdminRequest_RoleIDPtr_Set(t *testing.T) {
	roleID := utils.FlexInt(3)
	req := CreateAdminRequest{RoleID: &roleID}
	ptr := req.RoleIDPtr()
	if ptr == nil || *ptr != 3 {
		t.Errorf("expected RoleIDPtr to return 3, got %v", ptr)
	}
}

func TestUpdateAdminRequest_Valid(t *testing.T) {
	req := UpdateAdminRequest{Email: "a@b.com", Name: "Admin"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestUpdateAdminRequest_EmptyPicAllowed(t *testing.T) {
	// Update tidak memaksa pic/pic_client diisi (beda dari Create yang
	// Normalize()-nya default ke "no") - string kosong harus tetap lolos.
	req := UpdateAdminRequest{Email: "a@b.com", Name: "Admin", Pic: "", PicClient: ""}
	if err := req.Validate(); err != nil {
		t.Errorf("expected empty pic/pic_client to be allowed on update, got error: %v", err)
	}
}

func TestUpdateAdminRequest_InvalidPicValue(t *testing.T) {
	req := UpdateAdminRequest{Email: "a@b.com", Name: "Admin", Pic: "invalid"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for invalid pic value")
	}
}
