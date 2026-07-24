package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type AccessHandler struct{ repo *repository.AccessRepo }

func NewAccessHandler(repo *repository.AccessRepo) *AccessHandler { return &AccessHandler{repo: repo} }

func (h *AccessHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data akses")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewAccessResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

// GET /api/access/module/{module}
func (h *AccessHandler) ListByModule(w http.ResponseWriter, r *http.Request) {
	module := chi.URLParam(r, "module")
	data, err := h.repo.ListByModule(r.Context(), module)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data akses")
		return
	}
	utils.OK(w, "Fetch success", dto.NewAccessResponseList(data))
}

func (h *AccessHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	data, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Akses tidak ditemukan")
		return
	}
	utils.OK(w, "Fetch success", dto.NewAccessResponse(*data))
}

func (h *AccessHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.AccessRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := h.repo.Create(r.Context(), req.Title, req.Module, req.Slug)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat akses")
		return
	}
	utils.Created(w, "Akses berhasil dibuat", map[string]any{"access_id": id})
}

func (h *AccessHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AccessRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.Update(r.Context(), id, req.Title, req.Module, req.Slug); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui akses")
		return
	}
	utils.OK(w, "Akses berhasil diperbarui", nil)
}

func (h *AccessHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus akses")
		return
	}
	utils.OK(w, "Akses berhasil dihapus", nil)
}