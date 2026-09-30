package dto

import (
	"errors"

	"rms-backend/internal/utils"
)

// LinkQuotationRequest - payload POST /api/po/{id}/link-quotation (§4.3).
type LinkQuotationRequest struct {
	QuotationID utils.FlexInt `json:"quotation_id"`
}

func (r LinkQuotationRequest) Validate() error {
	if int(r.QuotationID) == 0 {
		return errors.New("quotation_id wajib diisi")
	}
	return nil
}

// UnlinkQuotationRequest - payload POST /api/po/{id}/unlink-quotation (§4.4).
// Notes wajib diisi karena dicatat sebagai audit trail permanen di t_pr_comment.
type UnlinkQuotationRequest struct {
	QuotationID utils.FlexInt `json:"quotation_id"`
	Notes       string        `json:"notes"`
}

func (r UnlinkQuotationRequest) Validate() error {
	if int(r.QuotationID) == 0 {
		return errors.New("quotation_id wajib diisi")
	}
	if r.Notes == "" {
		return errors.New("alasan pencabutan link (notes) wajib diisi")
	}
	return nil
}