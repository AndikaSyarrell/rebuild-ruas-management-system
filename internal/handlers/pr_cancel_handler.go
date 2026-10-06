package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

const (
	maxCancelReasonRunes = 1200 
	maxCancelNotesRunes  = 500
)

var validCancelStatusFilter = map[string]bool{"pending": true, "approved": true, "rejected": true}

type PRCancelHandler struct {
	service    *service.PRCancelService
	repo       *repository.PRCancelRepo
	prRepo     *repository.PRRepo
	accessRepo *repository.AccessRepo
	uploadDir  string
}

func NewPRCancelHandler(svc *service.PRCancelService, repo *repository.PRCancelRepo, prRepo *repository.PRRepo,
	accessRepo *repository.AccessRepo, uploadDir string) *PRCancelHandler {
	return &PRCancelHandler{service: svc, repo: repo, prRepo: prRepo, accessRepo: accessRepo, uploadDir: uploadDir}
}

// POST /api/pr/{id}/cancel - batal langsung (hanya bila belum lolos checker)
func (h *PRCancelHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CancelPRRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	err := h.service.CancelDirect(r.Context(), id, adminID, req.Notes)
	switch {
	case err == nil:
		utils.OK(w, "Purchase request berhasil dibatalkan", nil)
	case errors.Is(err, service.ErrNotFound):
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
	case errors.Is(err, service.ErrPRCancelNotOwner):
		utils.Error(w, http.StatusForbidden, "Anda hanya dapat membatalkan purchase request milik sendiri")
	case errors.Is(err, service.ErrPRAlreadyFinalized):
		utils.Error(w, http.StatusConflict, "Purchase request sudah pada status final dan tidak dapat dibatalkan")
	case errors.Is(err, service.ErrPRCancelRequiresRequest):
		// code ini dipakai frontend untuk membuka form request pembatalan
		utils.JSON(w, http.StatusConflict, false,
			"Purchase request sudah disetujui checker. Ajukan request pembatalan dengan berita acara dan ringkasan alasan.",
			map[string]any{"code": "cancel_request_required"})
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal membatalkan purchase request")
	}
}

// POST /api/pr/{id}/cancel-request  (multipart/form-data: file = berita acara, reason)
func (h *PRCancelHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	prID, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Ukuran file maksimal 10MB")
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		utils.Error(w, http.StatusBadRequest, "Ringkasan alasan pembatalan wajib diisi")
		return
	}
	if utf8.RuneCountInString(reason) > maxCancelReasonRunes {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("Ringkasan alasan maksimal %d karakter", maxCancelReasonRunes))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "File berita acara wajib diunggah")
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

	if err := os.MkdirAll(filepath.Join(h.uploadDir, "pr_cancel"), 0o755); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyiapkan penyimpanan")
		return
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	relPath := filepath.Join("uploads", "pr_cancel", filename)
	fullPath := filepath.Join(h.uploadDir, "pr_cancel", filename)

	out, err := os.Create(fullPath)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		os.Remove(fullPath)
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}
	if err := out.Close(); err != nil {
		os.Remove(fullPath)
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}

	id, err := h.service.CreateRequest(r.Context(), prID, adminID, reason, filepath.Base(header.Filename), relPath)
	if err != nil {
		os.Remove(fullPath) // jangan tinggalkan file yatim bila request ditolak
		switch {
		case errors.Is(err, service.ErrNotFound):
			utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
		case errors.Is(err, service.ErrPRCancelNotOwner):
			utils.Error(w, http.StatusForbidden, "Anda hanya dapat mengajukan pembatalan untuk purchase request milik sendiri")
		case errors.Is(err, service.ErrPRAlreadyFinalized):
			utils.Error(w, http.StatusConflict, "Purchase request sudah pada status final dan tidak dapat dibatalkan")
		case errors.Is(err, service.ErrPRCancelRequestNotRequired):
			utils.JSON(w, http.StatusConflict, false,
				"Purchase request ini masih dapat dibatalkan langsung, tidak perlu request pembatalan.",
				map[string]any{"code": "cancel_direct_allowed"})
		case errors.Is(err, repository.ErrCancelRequestAlreadyPending):
			utils.Error(w, http.StatusConflict, "Sudah ada request pembatalan yang menunggu review finance")
		default:
			utils.Error(w, http.StatusInternalServerError, "Gagal mengajukan pembatalan purchase request")
		}
		return
	}
	utils.Created(w, "Request pembatalan berhasil diajukan, menunggu review finance", map[string]any{"cancel_id": id})
}

// GET /api/pr/{id}/cancel-requests - riwayat request untuk satu PR (siapa pun yang boleh melihat PR-nya)
func (h *PRCancelHandler) ListByPR(w http.ResponseWriter, r *http.Request) {
	prID, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
		adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	pr, err := h.prRepo.GetByID(r.Context(), prID)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "Purchase request tidak ditemukan")
		return
	}
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data purchase request")
		return
	}
	if pr.RefAdmin != adminID {
		isFinance, err := h.accessRepo.HasAccess(r.Context(), adminID, "finance")
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa hak akses")
			return
		}
		if !isFinance {
			utils.Error(w, http.StatusForbidden, "Anda tidak memiliki akses ke request pembatalan ini")
			return
		}
	}
	data, err := h.repo.ListByPR(r.Context(), prID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil request pembatalan")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPRCancelRequestResponseList(data))
}

// GET /api/pr/cancel-requests?status=&page=&item=  (finance)
func (h *PRCancelHandler) ListForFinance(w http.ResponseWriter, r *http.Request) {
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	switch {
	case status == "":
		status = "pending"
	case status == "all":
		status = ""
	case !validCancelStatusFilter[status]:
		utils.Error(w, http.StatusBadRequest, "Status harus salah satu dari: pending, approved, rejected, all")
		return
	}
	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), status, p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar request pembatalan")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewPRCancelRequestResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

// POST /api/pr/cancel-requests/{cancelId}/approve  (finance) - langsung membatalkan PR
func (h *PRCancelHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.review(w, r, true)
}

// POST /api/pr/cancel-requests/{cancelId}/reject  (finance) - notes wajib
func (h *PRCancelHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.review(w, r, false)
}

func (h *PRCancelHandler) review(w http.ResponseWriter, r *http.Request, approve bool) {
	cancelID, ok := utils.ParseIDParam(w, chi.URLParam(r, "cancelId"))
	if !ok {
		return
	}
	var req dto.PRCancelReviewRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if !approve && req.Notes == "" {
		utils.Error(w, http.StatusBadRequest, "Alasan penolakan wajib diisi")
		return
	}
	if utf8.RuneCountInString(req.Notes) > maxCancelNotesRunes {
		utils.Error(w, http.StatusBadRequest, fmt.Sprintf("Catatan maksimal %d karakter", maxCancelNotesRunes))
		return
	}
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	var err error
	if approve {
		err = h.service.Approve(r.Context(), cancelID, adminID, req.Notes)
	} else {
		err = h.service.Reject(r.Context(), cancelID, adminID, req.Notes)
	}
	switch {
	case err == nil && approve:
		utils.OK(w, "Request disetujui, purchase request berhasil dibatalkan", nil)
	case err == nil:
		utils.OK(w, "Request pembatalan ditolak", nil)
	case errors.Is(err, service.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		utils.Error(w, http.StatusNotFound, "Request pembatalan tidak ditemukan")
	case errors.Is(err, service.ErrCancelReviewerIsRequester):
		utils.Error(w, http.StatusForbidden, "Anda tidak dapat mereview request pembatalan milik sendiri")
	case errors.Is(err, repository.ErrCancelRequestNotPending):
		utils.Error(w, http.StatusConflict, "Request pembatalan ini sudah direview")
	case errors.Is(err, repository.ErrPRNotCancellable):
		utils.Error(w, http.StatusConflict, "Status purchase request sudah tidak memungkinkan pembatalan")
	case errors.Is(err, repository.ErrPRHasPaidPayment):
		utils.Error(w, http.StatusConflict, "Purchase request memiliki pembayaran yang sudah lunas, tidak dapat dibatalkan")
	default:
		utils.Error(w, http.StatusInternalServerError, "Gagal memproses request pembatalan")
	}
}

// GET /api/pr/cancel-requests/{cancelId}/document - hanya pengaju atau finance
func (h *PRCancelHandler) Document(w http.ResponseWriter, r *http.Request) {
	cancelID, ok := utils.ParseIDParam(w, chi.URLParam(r, "cancelId"))
	if !ok {
		return
	}
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	req, err := h.repo.GetByID(r.Context(), cancelID)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "Request pembatalan tidak ditemukan")
		return
	}
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil request pembatalan")
		return
	}
	if req.RefAdmin != adminID {
		isFinance, err := h.accessRepo.HasAccess(r.Context(), adminID, "finance")
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa hak akses")
			return
		}
		if !isFinance {
			utils.Error(w, http.StatusForbidden, "Anda tidak memiliki akses ke dokumen ini")
			return
		}
	}

	fullPath := filepath.Join(h.uploadDir, "..", req.DocumentPath)
	f, err := os.Open(fullPath)
	if err != nil {
		log.Printf("dokumen pembatalan id=%d gagal buka file path=%s: %v", cancelID, fullPath, err)
		utils.Error(w, http.StatusNotFound, "File dokumen tidak ditemukan di server")
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil || fi.Size() == 0 {
		log.Printf("dokumen pembatalan id=%d file tidak valid path=%s err=%v", cancelID, fullPath, err)
		utils.Error(w, http.StatusInternalServerError, "File dokumen kosong atau rusak di server")
		return
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(req.DocumentPath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": req.DocumentName}))
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("dokumen pembatalan id=%d gagal streaming: %v", cancelID, err)
	}
}