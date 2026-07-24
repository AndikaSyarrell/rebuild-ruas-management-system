package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type PpnHandler struct{ repo *repository.PpnRepo }

func NewPpnHandler(repo *repository.PpnRepo) *PpnHandler { return &PpnHandler{repo: repo} }

// GET /api/ppn
func (h *PpnHandler) List(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.List(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data PPN")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPpnResponseList(data))
}

// GET /api/ppn/current
func (h *PpnHandler) Current(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.GetCurrent(r.Context())
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Tarif PPN belum diatur")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPpnResponse(*data))
}

// GET /api/ppn/{id}  - detail untuk form edit (menutup Feature Gap §8)
func (h *PpnHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	data, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Data PPN tidak ditemukan")
		return
	}
	utils.OK(w, "Fetch success", dto.NewPpnResponse(*data))
}

func (h *PpnHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.PpnRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.Update(r.Context(), id, float64(req.Value)); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui PPN")
		return
	}
	utils.OK(w, "PPN berhasil diperbarui", nil)
}

func (h *PpnHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.PpnRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := h.repo.Create(r.Context(), float64(req.Value))
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat PPN")
		return
	}
	utils.Created(w, "PPN berhasil dibuat", map[string]any{"ppn_id": id})
}