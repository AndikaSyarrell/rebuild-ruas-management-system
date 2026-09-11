package dto

import (
	"errors"
	"strings"

	"rms-backend/internal/utils"
)

// CreateAdminRequest adalah payload pembuatan admin baru.
type CreateAdminRequest struct {
	Email     string         `json:"email"`
	Name      string         `json:"username"`
	RoleID    *utils.FlexInt `json:"role_id"`
	RegionID  utils.FlexInt  `json:"region_id"`
	DivisionID *utils.FlexInt `json:"division_id"` // BARU - opsional, boleh nil
	Pic       string         `json:"pic"`        // "yes" | "no"
	PicClient string         `json:"pic_client"` // "yes" | "no"
}

var validYesNo = map[string]bool{"yes": true, "no": true, "": true} // "" ditangani default di handler

func (r *CreateAdminRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Name = strings.TrimSpace(r.Name)
	if r.Pic == "" {
		r.Pic = "no"
	}
	if r.PicClient == "" {
		r.PicClient = "no"
	}
}

func (r CreateAdminRequest) DivisionIDPtr() *int {
	if r.DivisionID == nil {
		return nil
	}
	v := int(*r.DivisionID)
	return &v
}

func (r CreateAdminRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email wajib diisi")
	}
	if r.Name == "" {
		return errors.New("nama wajib diisi")
	}
	if int(r.RegionID) == 0 {
		return errors.New("region wajib dipilih")
	}
	if !validYesNo[r.Pic] {
		return errors.New(`nilai "pic" harus "yes" atau "no"`)
	}
	if !validYesNo[r.PicClient] {
		return errors.New(`nilai "pic_client" harus "yes" atau "no"`)
	}
	return nil
}

func (r CreateAdminRequest) RoleIDPtr() *int {
	if r.RoleID == nil {
		return nil
	}
	v := int(*r.RoleID)
	return &v
}

// UpdateAdminRequest adalah payload update data admin (tanpa ganti password).
type UpdateAdminRequest struct {
	Email     string         `json:"email"`
	Name      string         `json:"username"`
	RoleID    *utils.FlexInt `json:"role_id"`
	RegionID  utils.FlexInt  `json:"region_id"`
	DivisionID *utils.FlexInt `json:"division_id"` // BARU
	Pic       string         `json:"pic"`
	PicClient string         `json:"pic_client"`
}

func (r UpdateAdminRequest) DivisionIDPtr() *int {
	if r.DivisionID == nil {
		return nil
	}
	v := int(*r.DivisionID)
	return &v
}

func (r *UpdateAdminRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Name = strings.TrimSpace(r.Name)
}

func (r UpdateAdminRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email wajib diisi")
	}
	if r.Name == "" {
		return errors.New("nama wajib diisi")
	}
	if r.Pic != "" && !validYesNo[r.Pic] {
		return errors.New(`nilai "pic" harus "yes" atau "no"`)
	}
	if r.PicClient != "" && !validYesNo[r.PicClient] {
		return errors.New(`nilai "pic_client" harus "yes" atau "no"`)
	}
	return nil
}

func (r UpdateAdminRequest) RoleIDPtr() *int {
	if r.RoleID == nil {
		return nil
	}
	v := int(*r.RoleID)
	return &v
}