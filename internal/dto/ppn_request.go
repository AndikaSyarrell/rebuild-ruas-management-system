package dto

import (
	"errors"

	"rms-backend/internal/utils"
)

// PpnRequest adalah payload create/update tarif PPN.
type PpnRequest struct {
	Value utils.FlexFloat `json:"value"`
}

func (r PpnRequest) Validate() error {
	v := float64(r.Value)
	if v < 0 || v > 100 {
		return errors.New("nilai PPN harus di antara 0 dan 100")
	}
	return nil
}