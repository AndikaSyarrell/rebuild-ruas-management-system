package dto

import (
	"time"

	"rms-backend/internal/models"
)

type ResponsibleResponse struct {
	ID         int    `json:"responsible_id"`
	Name       string `json:"responsible_name"`
	CoaCode    string `json:"responsible_coa_code"`
	Status     string `json:"responsible_status"`
	CreateDate string `json:"responsible_create_date"`
}

func NewResponsibleResponse(m models.Responsible) ResponsibleResponse {
	return ResponsibleResponse{
		ID: m.ID, Name: m.Name, CoaCode: m.CoaCode, Status: m.Status,
		CreateDate: m.CreateDate.Format(time.RFC3339),
	}
}

func NewResponsibleResponseList(list []models.Responsible) []ResponsibleResponse {
	out := make([]ResponsibleResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewResponsibleResponse(m))
	}
	return out
}