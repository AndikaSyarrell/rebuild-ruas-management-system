package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type PpnHandler struct{ repo *repository.PpnRepo }

func NewPpnHandler(repo *repository.PpnRepo) *PpnHandler { return &PpnHandler{repo: repo} }

// GET /api/ppn  - daftar tarif (histori)
func (h *PpnHandler) List(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.List(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data PPN")
		return
	}
	utils.OK(w, "Fetch success", data)
}

// GET /api/ppn/current  - tarif yang sedang berlaku, dipakai form buat PO baru
func (h *PpnHandler) Current(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.GetCurrent(r.Context())
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Tarif PPN belum diatur")
		return
	}
	utils.OK(w, "Fetch success", data)
}

type ppnRequest struct {
	Value float64 `json:"value"`
}

func (h *PpnHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var req ppnRequest
	if err := decodeJSON(r, &req); err != nil || req.Value < 0 {
		utils.Error(w, http.StatusBadRequest, "Nilai PPN tidak valid")
		return
	}
	if err := h.repo.Update(r.Context(), id, req.Value); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui PPN")
		return
	}
	utils.OK(w, "PPN berhasil diperbarui", nil)
}

func (h *PpnHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ppnRequest
	if err := decodeJSON(r, &req); err != nil || req.Value < 0 {
		utils.Error(w, http.StatusBadRequest, "Nilai PPN tidak valid")
		return
	}
	id, err := h.repo.Create(r.Context(), req.Value)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat PPN")
		return
	}
	utils.Created(w, "PPN berhasil dibuat", map[string]any{"ppn_id": id})
}
