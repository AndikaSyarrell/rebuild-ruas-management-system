package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type UnitHandler struct{ repo *repository.UnitRepo }

func NewUnitHandler(repo *repository.UnitRepo) *UnitHandler { return &UnitHandler{repo: repo} }

func (h *UnitHandler) Select(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListSelect(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data unit")
		return
	}
	utils.OK(w, "Fetch success", dto.NewUnitResponseList(data))
}

func (h *UnitHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data unit")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewUnitResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

func (h *UnitHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	data, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Unit tidak ditemukan")
		return
	}
	utils.OK(w, "Fetch success", dto.NewUnitResponse(*data))
}

func (h *UnitHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.TitleRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, "Judul unit wajib diisi")
		return
	}
	id, err := h.repo.Create(r.Context(), req.Title)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat unit")
		return
	}
	utils.Created(w, "Unit berhasil dibuat", map[string]any{"unit_id": id})
}

func (h *UnitHandler) Update(w http.ResponseWriter, r *http.Request) {
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
		utils.Error(w, http.StatusBadRequest, "Judul unit wajib diisi")
		return
	}
	if err := h.repo.Update(r.Context(), id, req.Title); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui unit")
		return
	}
	utils.OK(w, "Unit berhasil diperbarui", nil)
}

func (h *UnitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus unit")
		return
	}
	utils.OK(w, "Unit berhasil dihapus", nil)
}