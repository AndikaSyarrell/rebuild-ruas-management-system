package dto

import (
	"time"

	"rms-backend/internal/models"
)

type DocumentResponse struct {
	ID         int    `json:"document_id"`
	RefPO      string `json:"document_ref_po"`
	Title      string `json:"document_title"`
	File       string `json:"document_file"`
	CreateDate string `json:"document_create_date"`
}

func NewDocumentResponse(m models.PODocument) DocumentResponse {
	return DocumentResponse{
		ID: m.ID, RefPO: m.RefPO, Title: m.Title, File: m.File,
		CreateDate: m.CreateDate.Format(time.RFC3339),
	}
}

func NewDocumentResponseList(list []models.PODocument) []DocumentResponse {
	out := make([]DocumentResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewDocumentResponse(m))
	}
	return out
}