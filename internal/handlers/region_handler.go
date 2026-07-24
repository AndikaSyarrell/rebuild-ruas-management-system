package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type RegionHandler struct {
	repo *repository.RegionRepo
}

func NewRegionHandler(repo *repository.RegionRepo) *RegionHandler {
	return &RegionHandler{repo: repo}
}

// GET /api/regions/select
func (h *RegionHandler) Select(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.List(r.Context(), "")
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data region")
		return
	}
	utils.OK(w, "Fetch success", dto.NewRegionResponseList(data))
}

// GET /api/regions
func (h *RegionHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data region")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewRegionResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

// GET /api/regions/{id}
func (h *RegionHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	data, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Region tidak ditemukan")
		return
	}
	utils.OK(w, "Fetch success", dto.NewRegionResponse(*data))
}

// POST /api/regions
func (h *RegionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.TitleRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, "Judul region wajib diisi")
		return
	}
	id, err := h.repo.Create(r.Context(), req.Title)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat region")
		return
	}
	utils.Created(w, "Region berhasil dibuat", map[string]any{"region_id": id})
}

// PUT /api/regions/{id}
func (h *RegionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.TitleRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, "Judul region wajib diisi")
		return
	}
	if err := h.repo.Update(r.Context(), id, req.Title); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui region")
		return
	}
	utils.OK(w, "Region berhasil diperbarui", nil)
}

// DELETE /api/regions/{id}
func (h *RegionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus region (kemungkinan masih dipakai data lain)")
		return
	}
	utils.OK(w, "Region berhasil dihapus", nil)
}