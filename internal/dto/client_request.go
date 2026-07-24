package dto

import (
	"errors"
	"strings"

	"rms-backend/internal/utils"
)

// ClientRequest adalah payload create/update data klien.
type ClientRequest struct {
	Name     string        `json:"name"`
	Email    string        `json:"email"`
	Phone    string        `json:"phone"`
	Address  string        `json:"address"`
	RegionID utils.FlexInt `json:"region_id"`
}

func (r *ClientRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Phone = strings.TrimSpace(r.Phone)
	r.Address = strings.TrimSpace(r.Address)
}

func (r ClientRequest) Validate() error {
	if r.Name == "" {
		return errors.New("nama klien wajib diisi")
	}
	if r.Email == "" {
		return errors.New("email klien wajib diisi")
	}
	if int(r.RegionID) == 0 {
		return errors.New("region wajib dipilih")
	}
	return nil
}