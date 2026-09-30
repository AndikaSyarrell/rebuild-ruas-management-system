package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"log"
	"mime"
	"strconv"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type PRDocumentHandler struct {
	repo      *repository.PRDocumentRepo
	prRepo    *repository.PRRepo
	uploadDir string
}

func NewPRDocumentHandler(repo *repository.PRDocumentRepo, prRepo *repository.PRRepo, uploadDir string) *PRDocumentHandler {
	return &PRDocumentHandler{repo: repo, prRepo: prRepo, uploadDir: uploadDir}
}

var allowedPRDocExt = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".jpg": true, ".jpeg": true, ".png": true,
}

var allowedPRDocMIME = map[string]bool{
	"application/pdf":     true,
	"image/jpeg":          true,
	"image/png":           true,
	"application/x-cfb":   true, // .doc lama
	"application/msword":  true,
	"application/zip":     true, // .docx (OOXML)
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

// validPRDocTypes membatasi document_type ke nilai yang dikenali sistem -
// "cost_control" WAJIB persis string ini karena dicek langsung oleh
// PRApprovalService.SubmitForApproval (lihat costControlDocType).
var validPRDocTypes = map[string]bool{
	"cost_control": true,
	"quotation":    true,
	"invoice":      true,
	"contract":     true,
	"other":        true,
}

// GET /api/pr/{id}/documents
func (h *PRDocumentHandler) ListByPR(w http.ResponseWriter, r *http.Request) {
	prID, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !requirePRVisible(w, r, h.prRepo, prID) {
		return
	}
	if !ok {
		return
	}
	data, err := h.repo.ListByPR(r.Context(), prID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar dokumen PR")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPRDocumentResponseList(data))
}

// POST /api/pr/{id}/documents  (multipart/form-data: file, document_type)
func (h *PRDocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	prID, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	pr, err := h.prRepo.GetByID(r.Context(), prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data purchase request")
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	// Hanya requester pemilik PR yang boleh melampirkan dokumen - konsisten
	// dengan PRHandler.Update yang membatasi hal serupa.
	if pr.RefAdmin != adminID {
		utils.Error(w, http.StatusForbidden, "Anda hanya dapat melampirkan dokumen pada purchase request milik sendiri")
		return
	}

	docType := strings.TrimSpace(strings.ToLower(r.FormValue("document_type")))
	if docType == "" {
		docType = "other"
	}
	if !validPRDocTypes[docType] {
		utils.Error(w, http.StatusBadRequest, "document_type tidak dikenali (gunakan: cost_control, quotation, invoice, contract, other)")
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
	if !allowedPRDocExt[ext] {
		utils.Error(w, http.StatusBadRequest, "Format file harus pdf, doc, docx, jpg, jpeg, atau png")
		return
	}
	if err := utils.ValidateFileContent(file, allowedPRDocMIME); err != nil {
		utils.Error(w, http.StatusBadRequest, "Isi file tidak sesuai dengan format dokumen yang diklaim")
		return
	}

	if err := os.MkdirAll(filepath.Join(h.uploadDir, "pr_document"), 0o755); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyiapkan penyimpanan")
		return
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	relPath := filepath.Join("uploads", "pr_document", filename)
	fullPath := filepath.Join(h.uploadDir, "pr_document", filename)

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

	id, err := h.repo.Insert(r.Context(), prID, docType, header.Filename, relPath, adminID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan metadata dokumen")
		return
	}

	utils.Created(w, "Dokumen berhasil diunggah", map[string]any{"document_id": id, "file": relPath})
}

// DELETE /api/pr/{id}/documents/{docId}
func (h *PRDocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	docID, ok := utils.ParseIDParam(w, chi.URLParam(r, "docId"))
	if !ok {
		return
	}

	doc, err := h.repo.GetByID(r.Context(), docID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Dokumen tidak ditemukan")
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	if doc.RefAdmin != adminID {
		utils.Error(w, http.StatusForbidden, "Anda hanya dapat menghapus dokumen yang Anda unggah sendiri")
		return
	}

	if err := h.repo.Delete(r.Context(), docID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus dokumen")
		return
	}
	_ = os.Remove(filepath.Join(h.uploadDir, "..", doc.FilePath))

	utils.OK(w, "Dokumen berhasil dihapus", nil)
}

// GET /api/pr/{id}/documents/{docId}/download
// Memaksa browser mengunduh file (Content-Disposition: attachment).
func (h *PRDocumentHandler) Download(w http.ResponseWriter, r *http.Request) {
	h.serveFile(w, r, true)
}

// GET /api/pr/{id}/documents/{docId}/preview
// Menampilkan file langsung di browser bila memungkinkan (Content-Disposition: inline),
// cocok dipakai untuk preview PDF/gambar dari frontend (mis. <iframe>/<img>).
func (h *PRDocumentHandler) Preview(w http.ResponseWriter, r *http.Request) {
	h.serveFile(w, r, false)
}

// serveFile berisi logika bersama Download & Preview - satu-satunya beda
// adalah header Content-Disposition (attachment vs inline).
func (h *PRDocumentHandler) serveFile(w http.ResponseWriter, r *http.Request, forceDownload bool) {
	prID, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	docID, ok := utils.ParseIDParam(w, chi.URLParam(r, "docId"))
	if !ok {
		return
	}

	doc, err := h.repo.GetByID(r.Context(), docID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Dokumen tidak ditemukan")
		return
	}
	// Pastikan docId yang diminta memang milik prId di path - mencegah
	// akses dokumen PR lain lewat tebak-tebakan docId.
	if doc.RefPR != prID {
		utils.Error(w, http.StatusNotFound, "Dokumen tidak ditemukan pada purchase request ini")
		return
	}

	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	// Hanya admin yang terkait dengan PR ini (requester, approver di salah
	// satu round, atau finance yang mencatat/mengonfirmasi pembayaran) yang
	// boleh membuka isi dokumennya - konsisten dengan proteksi komentar PR.
	related, err := h.prRepo.IsAdminRelated(r.Context(), prID, adminID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa keterkaitan dengan purchase request")
		return
	}
	if !related {
		utils.Error(w, http.StatusForbidden, "Anda tidak terkait dengan purchase request ini (bukan requester, approver, atau finance terkait)")
		return
	}

	fullPath := filepath.Join(h.uploadDir, "..", doc.FilePath)
	f, err := os.Open(fullPath)
	if err != nil {
		log.Printf("akses dokumen PR id=%d gagal buka file path=%s: %v", docID, fullPath, err)
		utils.Error(w, http.StatusNotFound, "File dokumen tidak ditemukan di server")
		return
	}
	defer f.Close()

	// Stat dulu sebelum streaming - supaya kalau file kosong/rusak, kita
	// masih bisa kirim status error yang benar (sebelum header 200 terkirim).
	fi, err := f.Stat()
	if err != nil {
		log.Printf("akses dokumen PR id=%d gagal stat file path=%s: %v", docID, fullPath, err)
		utils.Error(w, http.StatusInternalServerError, "Gagal membaca informasi file")
		return
	}
	if fi.Size() == 0 {
		log.Printf("akses dokumen PR id=%d file kosong (0 bytes) path=%s", docID, fullPath)
		utils.Error(w, http.StatusInternalServerError, "File dokumen kosong atau rusak di server")
		return
	}

	ext := strings.ToLower(filepath.Ext(doc.FilePath))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	disposition := "inline"
	if forceDownload {
		disposition = "attachment"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, doc.FileName))

	written, err := io.Copy(w, f)
	if err != nil {
		log.Printf("akses dokumen PR id=%d gagal streaming setelah %d/%d bytes: %v",
			docID, written, fi.Size(), err)
		return
	}
	if written != fi.Size() {
		log.Printf("akses dokumen PR id=%d MISMATCH jumlah byte: terkirim=%d, seharusnya=%d",
			docID, written, fi.Size())
	}
}