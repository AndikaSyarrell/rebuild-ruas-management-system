package dto

import (
	"time"

	"rms-backend/internal/models"
)

type AccessResponse struct {
	ID         int    `json:"access_id"`
	Title      string `json:"access_title"`
	Slug       string `json:"access_slug"`
	Module     string `json:"access_module"`
	Sort       int    `json:"access_sort"`
	CreateDate string `json:"access_create_date"`
}

func NewAccessResponse(m models.Access) AccessResponse {
	return AccessResponse{
		ID: m.ID, Title: m.Title, Slug: m.Slug, Module: m.Module,
		Sort: m.Sort, CreateDate: m.CreateDate.Format(time.RFC3339),
	}
}

func NewAccessResponseList(list []models.Access) []AccessResponse {
	out := make([]AccessResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewAccessResponse(m))
	}
	return out
}