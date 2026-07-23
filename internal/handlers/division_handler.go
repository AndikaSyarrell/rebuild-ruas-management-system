package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type DivisionHandler struct{ repo *repository.DivisionRepo }

func NewDivisionHandler(repo *repository.DivisionRepo) *DivisionHandler {
	return &DivisionHandler{repo: repo}
}

func (h *DivisionHandler) Select(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListSelect(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data divisi")
		return
	}
	utils.OK(w, "Fetch success", data)
}

func (h *DivisionHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data divisi")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", data, map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

func (h *DivisionHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	data, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Divisi tidak ditemukan")
		return
	}
	utils.OK(w, "Fetch success", data)
}

type divisionRequest struct {
	Title string `json:"title"`
}

func (h *DivisionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req divisionRequest
	if err := decodeJSON(r, &req); err != nil || req.Title == "" {
		utils.Error(w, http.StatusBadRequest, "Judul divisi wajib diisi")
		return
	}
	id, err := h.repo.Create(r.Context(), req.Title)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat divisi")
		return
	}
	utils.Created(w, "Divisi berhasil dibuat", map[string]any{"division_id": id})
}

func (h *DivisionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req divisionRequest
	if err := decodeJSON(r, &req); err != nil || req.Title == "" {
		utils.Error(w, http.StatusBadRequest, "Judul divisi wajib diisi")
		return
	}
	if err := h.repo.Update(r.Context(), id, req.Title); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui divisi")
		return
	}
	utils.OK(w, "Divisi berhasil diperbarui", nil)
}

func (h *DivisionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus divisi")
		return
	}
	utils.OK(w, "Divisi berhasil dihapus", nil)
}
