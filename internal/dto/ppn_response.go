package dto

import (
	"time"

	"rms-backend/internal/models"
)

type PpnResponse struct {
	ID         int     `json:"ppn_id"`
	Value      float64 `json:"ppn_value"`
	CreateDate string  `json:"ppn_create_date"`
}

func NewPpnResponse(m models.Ppn) PpnResponse {
	return PpnResponse{ID: m.ID, Value: m.Value, CreateDate: m.CreateDate.Format(time.RFC3339)}
}

func NewPpnResponseList(list []models.Ppn) []PpnResponse {
	out := make([]PpnResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewPpnResponse(m))
	}
	return out
}