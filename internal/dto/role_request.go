package dto

import (
	"errors"
	"strings"

	"rms-backend/internal/utils"
)

// RoleRequest adalah payload create/update role beserta daftar access_id
// yang dimiliki.
type RoleRequest struct {
	Title     string          `json:"title"`
	AccessIDs []utils.FlexInt `json:"access_ids"`
}

func (r *RoleRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
}

func (r RoleRequest) Validate() error {
	if r.Title == "" {
		return errors.New("judul role wajib diisi")
	}
	return nil
}

func (r RoleRequest) AccessIDInts() []int {
	out := make([]int, len(r.AccessIDs))
	for i, a := range r.AccessIDs {
		out[i] = int(a)
	}
	return out
}