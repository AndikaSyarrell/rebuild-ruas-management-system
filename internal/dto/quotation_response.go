package dto

import (
	"time"

	"rms-backend/internal/models"
	"rms-backend/internal/repository"
)

// QuotationCandidateResponse - satu baris hasil GET
// /api/po/{id}/quotation-candidates (§4.2): quotation yang belum ter-link,
// beserta daftar RFP anggota + status-nya, supaya admin bisa menilai
// BR-LINK-01 (minimal satu anggota completed) sebelum submit link.
type QuotationCandidateResponse struct {
	QuotationID int                       `json:"quotation_id"`
	QuotationNo string                    `json:"quotation_no"`
	CreateDate  string                    `json:"quotation_create_date"`
	Eligible    bool                      `json:"eligible"` // true bila >= 1 anggota berstatus completed (BR-LINK-01)
	Members     []QuotationMemberResponse `json:"members"`
	LinkedPO    string                    `json:"linked_po_no,omitempty"`
}

func NewQuotationCandidateResponse(c repository.QuotationCandidate) QuotationCandidateResponse {
	members := make([]QuotationMemberResponse, 0, len(c.Members))
	eligible := false
	for _, m := range c.Members {
		members = append(members, QuotationMemberResponse{PRID: m.PRID, RfpNo: m.RfpNo, Status: m.Status})
		if m.Status == "completed" {
			eligible = true
		}
	}
	return QuotationCandidateResponse{
		QuotationID: c.Quotation.ID,
		QuotationNo: c.Quotation.No,
		CreateDate:  c.Quotation.CreateDate.Format(time.RFC3339),
		Eligible:    eligible,
		Members:     members,
		LinkedPO:    c.Quotation.POOrderNum,
	}
}

func NewQuotationCandidateResponseList(list []repository.QuotationCandidate) []QuotationCandidateResponse {
	out := make([]QuotationCandidateResponse, 0, len(list))
	for _, c := range list {
		out = append(out, NewQuotationCandidateResponse(c))
	}
	return out
}

// LinkedQuotationResponse - satu entri pada perluasan GET /api/po/{id} (§4.5):
// quotation yang SUDAH ter-link ke PO ini, beserta ringkasan anggotanya.
type LinkedQuotationResponse struct {
	QuotationID   int                       `json:"quotation_id"`
	QuotationNo   string                    `json:"quotation_no"`
	LinkedBy      string                    `json:"quotation_link_ref_admin,omitempty"`
	LinkDate      string                    `json:"quotation_link_date,omitempty"`
	Members       []QuotationMemberResponse `json:"members"`
}

func NewLinkedQuotationResponse(q models.PRQuotation, members []repository.QuotationMember) LinkedQuotationResponse {
	linkedBy := ""
	if q.LinkRefAdmin != nil {
		linkedBy = *q.LinkRefAdmin
	}
	linkDate := ""
	if q.LinkDate != nil {
		linkDate = q.LinkDate.Format(time.RFC3339)
	}
	out := make([]QuotationMemberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, QuotationMemberResponse{PRID: m.PRID, RfpNo: m.RfpNo, Status: m.Status})
	}
	return LinkedQuotationResponse{
		QuotationID: q.ID, QuotationNo: q.No, LinkedBy: linkedBy, LinkDate: linkDate, Members: out,
	}
}