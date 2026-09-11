package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type ResponsibleHandler struct{ repo *repository.ResponsibleRepo }

func NewResponsibleHandler(repo *repository.ResponsibleRepo) *ResponsibleHandler {
	return &ResponsibleHandler{repo: repo}
}

// GET /api/responsibles/select - dropdown form PR, hanya yang berstatus aktif
func (h *ResponsibleHandler) Select(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListSelect(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data responsible")
		return
	}
	utils.OK(w, "Fetch success", dto.NewResponsibleResponseList(data))
}

// GET /api/responsibles?keyword=&page=&item=
func (h *ResponsibleHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	keyword := r.URL.Query().Get("keyword")
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage, keyword)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data responsible")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewResponsibleResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

func (h *ResponsibleHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	data, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Responsible tidak ditemukan")
		return
	}
	utils.OK(w, "Fetch success", dto.NewResponsibleResponse(*data))
}

func (h *ResponsibleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.ResponsibleRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	exists, err := h.repo.CoaCodeExists(r.Context(), req.CoaCode)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa kode CoA")
		return
	}
	if exists {
		utils.Error(w, http.StatusConflict, "Kode CoA sudah terdaftar")
		return
	}

	id, err := h.repo.Create(r.Context(), req.Name, req.CoaCode, req.Status)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat responsible")
		return
	}
	utils.Created(w, "Responsible berhasil dibuat", map[string]any{"responsible_id": id})
}

func (h *ResponsibleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.ResponsibleRequest
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
		utils.Error(w, http.StatusNotFound, "Responsible tidak ditemukan")
		return
	}
	if current.CoaCode != req.CoaCode {
		exists, err := h.repo.CoaCodeExists(r.Context(), req.CoaCode)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa kode CoA")
			return
		}
		if exists {
			utils.Error(w, http.StatusConflict, "Kode CoA sudah terdaftar")
			return
		}
	}

	if err := h.repo.Update(r.Context(), id, req.Name, req.CoaCode); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui responsible")
		return
	}
	utils.OK(w, "Responsible berhasil diperbarui", nil)
}

func (h *ResponsibleHandler) Activate(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "active")
}

func (h *ResponsibleHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "inactive")
}

func (h *ResponsibleHandler) setStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.repo.ChangeStatus(r.Context(), id, status); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui status responsible")
		return
	}
	utils.OK(w, "Status responsible berhasil diperbarui", nil)
}

func (h *ResponsibleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.ParseIDParam(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus responsible (kemungkinan masih dipakai purchase request lain)")
		return
	}
	utils.OK(w, "Responsible berhasil dihapus", nil)
}