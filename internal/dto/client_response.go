package dto

import (
	"time"

	"rms-backend/internal/models"
)

// ClientResponse adalah representasi klien yang AMAN dikirim ke client -
// TIDAK PERNAH menyertakan client_password, client_token, atau
// client_reset_code.
type ClientResponse struct {
	ID          int    `json:"client_id"`
	Name        string `json:"client_name"`
	Phone       string `json:"client_phone"`
	Email       string `json:"client_email"`
	Active      string `json:"client_active"`
	Address     string `json:"client_address"`
	RefRegion   int    `json:"client_ref_region"`
	RegionTitle string `json:"region_title,omitempty"`
	CreateDate  string `json:"client_create_date"`
}

func NewClientResponse(m models.Client) ClientResponse {
	return ClientResponse{
		ID:          m.ID,
		Name:        m.Name,
		Phone:       m.Phone,
		Email:       m.Email,
		Active:      m.Active,
		Address:     m.Address,
		RefRegion:   m.RefRegion,
		RegionTitle: m.RegionTitle,
		CreateDate:  m.CreateDate.Format(time.RFC3339),
	}
}

func NewClientResponseList(list []models.Client) []ClientResponse {
	out := make([]ClientResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewClientResponse(m))
	}
	return out
}

// ClientSelectResponse adalah representasi ringkas untuk dropdown/pencarian
// (endpoint Select) - hanya field yang dibutuhkan UI untuk memilih klien.
type ClientSelectResponse struct {
	ID    int    `json:"client_id"`
	Name  string `json:"client_name"`
	Email string `json:"client_email"`
	Phone string `json:"client_phone"`
	Address string `json:"client_address"`
}

func NewClientSelectResponse(m models.Client) ClientSelectResponse {
	return ClientSelectResponse{
		ID:    m.ID,
		Name:  m.Name,
		Email: m.Email,
		Phone: m.Phone,
		Address: m.Address,
	}
}

func NewClientSelectResponseList(list []models.Client) []ClientSelectResponse {
	out := make([]ClientSelectResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewClientSelectResponse(m))
	}
	return out
}