package dto

import (
	"time"

	"rms-backend/internal/models"
)

type ActivityResponse struct {
	ID         int    `json:"activity_id"`
	RefPO      string `json:"activity_ref_po"`
	Type       string `json:"activity_type"`
	Notes      string `json:"activity_notes,omitempty"`
	CreateDate string `json:"activity_create_date"`
	AdminName  string `json:"admin_name,omitempty"`
	AdminImg   string `json:"admin_img,omitempty"`
}

func NewActivityResponse(m models.Activity) ActivityResponse {
	notes := ""
	if m.Notes != nil {
		notes = *m.Notes
	}
	return ActivityResponse{
		ID: m.ID, RefPO: m.RefPO, Type: m.Type, Notes: notes,
		CreateDate: m.CreateDate.Format(time.RFC3339),
		AdminName:  m.AdminName, AdminImg: m.AdminImg,
	}
}

func NewActivityResponseList(list []models.Activity) []ActivityResponse {
	out := make([]ActivityResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewActivityResponse(m))
	}
	return out
}