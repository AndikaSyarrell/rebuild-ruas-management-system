package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type ActivityHandler struct{ repo *repository.ActivityRepo }

func NewActivityHandler(repo *repository.ActivityRepo) *ActivityHandler {
	return &ActivityHandler{repo: repo}
}

// GET /api/po/{id}/activities
func (h *ActivityHandler) ListByPO(w http.ResponseWriter, r *http.Request) {
	poID := chi.URLParam(r, "id")
	data, err := h.repo.ListByPO(r.Context(), poID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil aktivitas PO")
		return
	}
	utils.OK(w, "Fetch success", dto.NewActivityResponseList(data))
}

// GET /api/activities/dashboard
func (h *ActivityHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListDashboard(r.Context(), p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil aktivitas terbaru")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewActivityResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

type noteRequest struct {
	Note string `json:"note"`
}

// POST /api/po/{id}/notes-activity
func (h *ActivityHandler) AddNote(w http.ResponseWriter, r *http.Request) {
	poID := chi.URLParam(r, "id")
	var req noteRequest
	if err := decodeJSON(r, &req); err != nil || req.Note == "" {
		utils.Error(w, http.StatusBadRequest, "Catatan wajib diisi")
		return
	}
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	if _, err := h.repo.Insert(r.Context(), poID, adminID, "notes", req.Note); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan catatan")
		return
	}
	utils.Created(w, "Catatan berhasil disimpan", nil)
}