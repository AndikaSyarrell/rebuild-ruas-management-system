package dto

import (
	"time"

	"rms-backend/internal/models"
)

// RegionResponse
type RegionResponse struct {
	ID         int    `json:"region_id"`
	Title      string `json:"region_title"`
	CreateDate string `json:"region_create_date"`
}

func NewRegionResponse(m models.Region) RegionResponse {
	return RegionResponse{ID: m.ID, Title: m.Title, CreateDate: m.CreateDate.Format(time.RFC3339)}
}

func NewRegionResponseList(list []models.Region) []RegionResponse {
	out := make([]RegionResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewRegionResponse(m))
	}
	return out
}

// DivisionResponse
type DivisionResponse struct {
	ID         int    `json:"division_id"`
	Title      string `json:"division_title"`
	CreateDate string `json:"created_at"`
}

func NewDivisionResponse(m models.Division) DivisionResponse {
	return DivisionResponse{ID: m.ID, Title: m.Title, CreateDate: m.CreatedAt.Format(time.RFC3339)}
}

func NewDivisionResponseList(list []models.Division) []DivisionResponse {
	out := make([]DivisionResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewDivisionResponse(m))
	}
	return out
}

// UnitResponse
type UnitResponse struct {
	ID         int    `json:"unit_id"`
	Title      string `json:"unit_title"`
	CreateDate string `json:"unit_create_date"`
}

func NewUnitResponse(m models.Unit) UnitResponse {
	return UnitResponse{ID: m.ID, Title: m.Title, CreateDate: m.CreateDate.Format(time.RFC3339)}
}

func NewUnitResponseList(list []models.Unit) []UnitResponse {
	out := make([]UnitResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewUnitResponse(m))
	}
	return out
}