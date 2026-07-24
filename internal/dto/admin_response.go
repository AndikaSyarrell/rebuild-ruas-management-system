package dto

import (
	"time"

	"rms-backend/internal/models"
)

// AdminResponse adalah representasi admin yang AMAN dikirim ke client -
// TIDAK PERNAH menyertakan admin_password, admin_token, atau
// admin_reset_code, apapun yang terjadi pada models.Admin di masa depan.
// Ini yang membedakan DTO dari sekadar ToPublic(): field yang boleh keluar
// didaftarkan eksplisit di sini, bukan "semua field kecuali yang di-exclude".
type AdminResponse struct {
	ID          string `json:"admin_id"`
	Email       string `json:"admin_email"`
	Name        string `json:"admin_name"`
	Active      string `json:"admin_active"`
	Pic         string `json:"admin_pic"`
	PicClient   string `json:"admin_pic_client"`
	RefRole     *int   `json:"admin_ref_role"`
	RoleTitle   string `json:"role_title,omitempty"`
	RefRegion   int    `json:"admin_ref_region"`
	RegionTitle string `json:"region_title,omitempty"`
	Img         string `json:"admin_img,omitempty"`
	ImgThumb    string `json:"admin_img_thmb,omitempty"`
	CreateDate  string `json:"admin_create_date"`
}

func NewAdminResponse(m models.Admin) AdminResponse {
	img := ""
	if m.Img != nil {
		img = *m.Img
	}
	imgThumb := ""
	if m.ImgThumb != nil {
		imgThumb = *m.ImgThumb
	}
	return AdminResponse{
		ID:          m.ID,
		Email:       m.Email,
		Name:        m.Name,
		Active:      m.Active,
		Pic:         m.Pic,
		PicClient:   m.PicClient,
		RefRole:     m.RefRole,
		RoleTitle:   m.RoleTitle,
		RefRegion:   m.RefRegion,
		RegionTitle: m.RegionTitle,
		Img:         img,
		ImgThumb:    imgThumb,
		CreateDate:  m.CreateDate.Format(time.RFC3339),
	}
}

func NewAdminResponseList(list []models.Admin) []AdminResponse {
	out := make([]AdminResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewAdminResponse(m))
	}
	return out
}