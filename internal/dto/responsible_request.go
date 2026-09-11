package dto

import (
	"errors"
	"strings"
)

// ResponsibleRequest adalah payload create/update T_Responsible (referensi
// Chart of Accounts yang dipakai sebagai pr_ref_responsible pada modul PR).
type ResponsibleRequest struct {
	Name    string `json:"name"`
	CoaCode string `json:"coa_code"`
	Status  string `json:"status"` // "active" | "inactive", opsional saat create (default "active")
}

var validResponsibleStatus = map[string]bool{"active": true, "inactive": true, "": true}

func (r *ResponsibleRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.CoaCode = strings.TrimSpace(strings.ToUpper(r.CoaCode))
	r.Status = strings.TrimSpace(strings.ToLower(r.Status))
}

func (r ResponsibleRequest) Validate() error {
	if r.Name == "" {
		return errors.New("nama responsible wajib diisi")
	}
	if r.CoaCode == "" {
		return errors.New("kode CoA wajib diisi")
	}
	if !validResponsibleStatus[r.Status] {
		return errors.New(`nilai "status" harus "active" atau "inactive"`)
	}
	return nil
}