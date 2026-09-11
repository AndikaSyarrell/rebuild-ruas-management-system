package dto

import (
	"time"

	"rms-backend/internal/models"
)

type PRDocumentResponse struct {
	ID         int    `json:"document_id"`
	Type       string `json:"document_type"`
	FileName   string `json:"document_file_name"`
	FilePath   string `json:"document_file_path"`
	RefAdmin   string `json:"document_ref_admin"`
	AdminName  string `json:"admin_name,omitempty"`
	RefPR      int    `json:"document_ref_pr"`
	CreateDate string `json:"document_create_date"`
}

func NewPRDocumentResponse(m models.PRDocument) PRDocumentResponse {
	return PRDocumentResponse{
		ID: m.ID, Type: m.Type, FileName: m.FileName, FilePath: m.FilePath,
		RefAdmin: m.RefAdmin, AdminName: m.AdminName, RefPR: m.RefPR,
		CreateDate: m.CreateDate.Format(time.RFC3339),
	}
}

func NewPRDocumentResponseList(list []models.PRDocument) []PRDocumentResponse {
	out := make([]PRDocumentResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewPRDocumentResponse(m))
	}
	return out
}