package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"mime"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/repository"
	"rms-backend/internal/dto"
	"rms-backend/internal/utils"
)

type DocumentHandler struct {
	repo         *repository.DocumentRepo
	poRepo       *repository.PORepo
	activityRepo *repository.ActivityRepo
	uploadDir    string
}

func NewDocumentHandler(repo *repository.DocumentRepo, poRepo *repository.PORepo, activityRepo *repository.ActivityRepo, uploadDir string) *DocumentHandler {
	return &DocumentHandler{repo: repo, poRepo: poRepo, activityRepo: activityRepo, uploadDir: uploadDir}
}

var allowedDocExt = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
}

var allowedDocMIME = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	// .doc lama (OLE compound file)
	"application/x-cfb":   true,
	"application/msword":  true,
	// .docx (zip-based OOXML) - DetectContentType tidak baca isi internal ZIP,
	// jadi terdeteksi generik sebagai zip.
	"application/zip": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

// GET /api/po/{id}/documents
func (h *DocumentHandler) ListByPO(w http.ResponseWriter, r *http.Request) {
	poID := chi.URLParam(r, "id")
	data, err := h.repo.ListByPO(r.Context(), poID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar dokumen")
		return
	}
	utils.OK(w, "Fetch success", dto.NewDocumentResponseList(data))
}

// POST /api/po/{id}/documents  (multipart/form-data: file, title)
func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	poID := chi.URLParam(r, "id")

	if _, err := h.poRepo.GetDetail(r.Context(), poID); err != nil {
		utils.Error(w, http.StatusNotFound, "PO tidak ditemukan")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Ukuran file maksimal 10MB")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "File wajib diunggah")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedDocExt[ext] {
		utils.Error(w, http.StatusBadRequest, "Format file harus pdf, doc, docx, jpg, jpeg, png, atau gif")
		return
	}

	// Validasi isi file (magic number) sesuai kategori ekstensi yang diklaim -
	// mencegah file executable/script yang di-rename ekstensinya lolos upload.
	if err := utils.ValidateFileContent(file, allowedDocMIME); err != nil {
		utils.Error(w, http.StatusBadRequest, "Isi file tidak sesuai dengan format dokumen yang diklaim")
		return
	}

	title := r.FormValue("title")
	if title == "" {
		title = header.Filename
	}

	if err := os.MkdirAll(filepath.Join(h.uploadDir, "document"), 0o755); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyiapkan penyimpanan")
		return
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	relPath := filepath.Join("uploads", "document", filename)
	fullPath := filepath.Join(h.uploadDir, "document", filename)

	out, err := os.Create(fullPath)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}

	docID, err := h.repo.Insert(r.Context(), poID, title, relPath)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan metadata dokumen")
		return
	}

	_ = h.poRepo.UpdateDocInfo(r.Context(), poID, "yes")

	if adminID, ok := actorFromContext(r.Context()); ok {
		_, _ = h.activityRepo.Insert(r.Context(), poID, adminID, "upload", "Mengunggah dokumen "+title)
	}

	utils.Created(w, "Dokumen berhasil diunggah", map[string]any{"document_id": docID, "file": relPath})
}

// DELETE /api/po/{poId}/documents/{docId}
func (h *DocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	poID := chi.URLParam(r, "id")
	docID, ok := utils.ParseIDParam(w, chi.URLParam(r, "docId"))
	if !ok {
		return
	}

	doc, err := h.repo.GetByID(r.Context(), docID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Dokumen tidak ditemukan")
		return
	}

	if err := h.repo.Delete(r.Context(), docID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus dokumen")
		return
	}
	_ = os.Remove(filepath.Join(h.uploadDir, "..", doc.File))

	remaining, _ := h.repo.CountByPO(r.Context(), poID)
	status := "no"
	if remaining > 0 {
		status = "yes"
	}
	_ = h.poRepo.UpdateDocInfo(r.Context(), poID, status)

	if adminID, ok := actorFromContext(r.Context()); ok {
		_, _ = h.activityRepo.Insert(r.Context(), poID, adminID, "delete", "Menghapus 1 dokumen dari daftar")
	}

	utils.OK(w, "Dokumen berhasil dihapus", nil)
}

func (h *DocumentHandler) Download(w http.ResponseWriter, r *http.Request) {
	docID, ok := utils.ParseIDParam(w, chi.URLParam(r, "docId"))
	if !ok {
		return
	}
	doc, err := h.repo.GetByID(r.Context(), docID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Dokumen tidak ditemukan")
		return
	}

	fullPath := filepath.Join(h.uploadDir, "..", doc.File)
	f, err := os.Open(fullPath)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "File dokumen tidak ditemukan di server")
		return
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(doc.File))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	// "inline" (bukan "attachment") - browser boleh preview langsung (PDF/gambar
	// tampil di tab), frontend yang menentukan apakah dibuka tab baru (preview)
	// atau dipaksa save-as (download) lewat atribut <a download> di sisi client.
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, doc.Title))
	io.Copy(w, f)
}