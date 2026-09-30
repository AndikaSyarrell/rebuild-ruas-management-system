package dto

import (
	"strings"
	"time"

	"rms-backend/internal/repository"
)

type PRCancelReviewRequest struct {
	Notes string `json:"notes"`
}

func (r *PRCancelReviewRequest) Normalize() { r.Notes = strings.TrimSpace(r.Notes) }

type PRCancelRequestResponse struct {
	ID              int     `json:"cancel_id"`
	PRID            int     `json:"pr_id"`
	RfpNo           string  `json:"rfp_no"`
	DescriptionItem string  `json:"pr_description_item"`
	RequestedAmount float64 `json:"pr_requested_amount"`
	PRStatus        string  `json:"pr_status"`
	RequesterID     string  `json:"requester_id"`
	RequesterName   string  `json:"requester_name,omitempty"`
	Reason          string  `json:"cancel_reason"`
	DocumentName    string  `json:"cancel_document_name"`
	Status          string  `json:"cancel_status"`
	ReviewerName    string  `json:"reviewer_name,omitempty"`
	ReviewNotes     string  `json:"cancel_review_notes,omitempty"`
	ReviewDate      string  `json:"cancel_review_date,omitempty"`
	CreateDate      string  `json:"cancel_create_date"`
}

func NewPRCancelRequestResponseList(list []repository.PRCancelRequestView) []PRCancelRequestResponse {
	out := make([]PRCancelRequestResponse, 0, len(list))
	for _, v := range list {
		notes := ""
		if v.ReviewNotes != nil {
			notes = *v.ReviewNotes
		}
		out = append(out, PRCancelRequestResponse{
			ID: v.ID, PRID: v.RefPR, RfpNo: v.RfpNo, DescriptionItem: v.DescriptionItem,
			RequestedAmount: v.RequestedAmount, PRStatus: v.PRStatus,
			RequesterID: v.RefAdmin, RequesterName: v.RequesterName,
			Reason: v.Reason, DocumentName: v.DocumentName, Status: v.Status,
			ReviewerName: v.ReviewerName, ReviewNotes: notes,
			ReviewDate: formatDateTime(v.ReviewDate), CreateDate: v.CreateDate.Format(time.RFC3339),
		})
	}
	return out
}