package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type RoleHandler struct {
	repo       *repository.RoleRepo
	accessRepo *repository.AccessRepo
}

func NewRoleHandler(repo *repository.RoleRepo, accessRepo *repository.AccessRepo) *RoleHandler {
	return &RoleHandler{repo: repo, accessRepo: accessRepo}
}

func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data role")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", data, map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

func (h *RoleHandler) Select(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListSelect(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data role")
		return
	}
	utils.OK(w, "Fetch success", data)
}

// GET /api/roles/{id}  - detail role beserta daftar access_id yang dimiliki
func (h *RoleHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	role, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Role tidak ditemukan")
		return
	}
	accessIDs, err := h.accessRepo.GetAccessIDsForRole(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil daftar akses role")
		return
	}
	utils.OK(w, "Fetch success", map[string]any{"role": role, "access_ids": accessIDs})
}

func slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = strings.Join(strings.Fields(s), "_")
	return s
}

type roleRequest struct {
	Title     string `json:"title"`
	AccessIDs []int  `json:"access_ids"`
}

// POST /api/roles
func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if err := decodeJSON(r, &req); err != nil || req.Title == "" {
		utils.Error(w, http.StatusBadRequest, "Judul role wajib diisi")
		return
	}
	slug := slugify(req.Title)
	id, err := h.repo.Create(r.Context(), req.Title, slug)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat role")
		return
	}
	if len(req.AccessIDs) > 0 {
		if err := h.accessRepo.ReplaceRoleAccess(r.Context(), int(id), req.AccessIDs); err != nil {
			utils.Error(w, http.StatusInternalServerError, "Role dibuat, namun gagal menyimpan daftar akses")
			return
		}
	}
	utils.Created(w, "Role berhasil dibuat", map[string]any{"role_id": id})
}

// PUT /api/roles/{id}
func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var req roleRequest
	if err := decodeJSON(r, &req); err != nil || req.Title == "" {
		utils.Error(w, http.StatusBadRequest, "Judul role wajib diisi")
		return
	}
	slug := slugify(req.Title)
	if err := h.repo.Update(r.Context(), id, req.Title, slug); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui role")
		return
	}
	if err := h.accessRepo.ReplaceRoleAccess(r.Context(), id, req.AccessIDs); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Role diperbarui, namun gagal menyimpan daftar akses")
		return
	}
	utils.OK(w, "Role berhasil diperbarui", nil)
}

func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus role")
		return
	}
	utils.OK(w, "Role berhasil dihapus", nil)
}
