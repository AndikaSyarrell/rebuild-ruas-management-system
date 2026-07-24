package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type ClientHandler struct{ repo *repository.ClientRepo }

func NewClientHandler(repo *repository.ClientRepo) *ClientHandler { return &ClientHandler{repo: repo} }

// GET /api/clients/select?keyword=
func (h *ClientHandler) Select(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	data, err := h.repo.Search(r.Context(), keyword)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data klien")
		return
	}
	utils.OK(w, "Fetch success", dto.NewClientSelectResponseList(data))
}

// GET /api/clients?region=&keyword=
func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	regionID := utils.AtoiDefault(r.URL.Query().Get("region"), 0)
	keyword := r.URL.Query().Get("keyword")

	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage, regionID, keyword)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data klien")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewClientResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

func (h *ClientHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	data, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Klien tidak ditemukan")
		return
	}
	utils.OK(w, "Fetch success", dto.NewClientResponse(*data))
}

func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.ClientRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	exists, err := h.repo.EmailExists(r.Context(), req.Email)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa email")
		return
	}
	if exists {
		utils.Error(w, http.StatusConflict, "Email sudah terdaftar")
		return
	}

	id, err := h.repo.Create(r.Context(), req.Name, req.Phone, req.Email, req.Address, int(req.RegionID))
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat klien")
		return
	}
	utils.Created(w, "Klien berhasil dibuat", map[string]any{"client_id": id})
}

func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.ClientRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	current, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Klien tidak ditemukan")
		return
	}
	if current.Email != req.Email {
		exists, err := h.repo.EmailExists(r.Context(), req.Email)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa email")
			return
		}
		if exists {
			utils.Error(w, http.StatusConflict, "Email sudah terdaftar")
			return
		}
	}

	if err := h.repo.Update(r.Context(), id, req.Name, req.Phone, req.Email, req.Address, int(req.RegionID)); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui klien")
		return
	}
	utils.OK(w, "Klien berhasil diperbarui", nil)
}

func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus klien")
		return
	}
	utils.OK(w, "Klien berhasil dihapus", nil)
}

func (h *ClientHandler) Activate(w http.ResponseWriter, r *http.Request) {
	h.setActive(w, r, "yes")
}

func (h *ClientHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	h.setActive(w, r, "no")
}

func (h *ClientHandler) setActive(w http.ResponseWriter, r *http.Request, active string) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.repo.ChangeActive(r.Context(), id, active); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui status klien")
		return
	}
	utils.OK(w, "Status klien berhasil diperbarui", nil)
}