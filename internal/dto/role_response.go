package dto

import (
	"time"

	"rms-backend/internal/models"
)

type RoleResponse struct {
	ID          int    `json:"role_id"`
	Title       string `json:"role_title"`
	Slug        string `json:"role_slug"`
	CreateDate  string `json:"role_create_date"`
	TotalAccess int    `json:"total_access,omitempty"`
}

func NewRoleResponse(m models.Role) RoleResponse {
	return RoleResponse{
		ID: m.ID, Title: m.Title, Slug: m.Slug,
		CreateDate: m.CreateDate.Format(time.RFC3339), TotalAccess: m.TotalAccess,
	}
}

func NewRoleResponseList(list []models.Role) []RoleResponse {
	out := make([]RoleResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewRoleResponse(m))
	}
	return out
}

// RoleDetailResponse membungkus detail role + daftar access_id yang dimiliki.
type RoleDetailResponse struct {
	Role      RoleResponse `json:"role"`
	AccessIDs []int        `json:"access_ids"`
}

func NewRoleDetailResponse(m models.Role, accessIDs []int) RoleDetailResponse {
	if accessIDs == nil {
		accessIDs = []int{}
	}
	return RoleDetailResponse{Role: NewRoleResponse(m), AccessIDs: accessIDs}
}