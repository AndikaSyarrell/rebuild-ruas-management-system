package dto

import (
	"testing"

	"rms-backend/internal/models"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

func TestLinkQuotationRequest_Validate(t *testing.T) {
	if err := (LinkQuotationRequest{}).Validate(); err == nil {
		t.Error("quotation_id 0 harus ditolak")
	}
	if err := (LinkQuotationRequest{QuotationID: utils.FlexInt(3)}).Validate(); err != nil {
		t.Errorf("quotation_id valid harus lolos: %v", err)
	}
}

func TestUnlinkQuotationRequest_RequiresNotes(t *testing.T) {
	if err := (UnlinkQuotationRequest{QuotationID: utils.FlexInt(3)}).Validate(); err == nil {
		t.Error("notes kosong harus ditolak (audit trail wajib)")
	}
}

func TestQuotationCandidate_EligibleOnlyWhenAnyCompleted(t *testing.T) {
	c := repository.QuotationCandidate{
		Quotation: models.PRQuotation{ID: 1, No: "Q-1"},
		Members:   []repository.QuotationMember{{PRID: 1, Status: "approved"}},
	}
	if NewQuotationCandidateResponse(c).Eligible {
		t.Error("tanpa anggota completed tidak boleh eligible")
	}
	c.Members = append(c.Members, repository.QuotationMember{PRID: 2, Status: "completed"})
	if !NewQuotationCandidateResponse(c).Eligible {
		t.Error("ada anggota completed harus eligible (BR-LINK-01)")
	}
}