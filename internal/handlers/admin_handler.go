package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"rms-backend/internal/dto"
	"rms-backend/internal/repository"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type AdminHandler struct {
	repo        *repository.AdminRepo
	mail        *service.MailService
	uploadDir   string
	frontendURL string
}

func NewAdminHandler(repo *repository.AdminRepo, mail *service.MailService, uploadDir, frontendURL string) *AdminHandler {
	return &AdminHandler{repo: repo, mail: mail, uploadDir: uploadDir, frontendURL: frontendURL}
}

func (h *AdminHandler) List(w http.ResponseWriter, r *http.Request) {
	p := utils.ParsePagination(r)
	data, total, err := h.repo.ListPaged(r.Context(), p.Page, p.PerPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data admin")
		return
	}
	utils.JSONMeta(w, http.StatusOK, true, "Fetch success", dto.NewAdminResponseList(data), map[string]any{
		"total_data": total, "total_page": utils.TotalPage(total, p.PerPage), "page": p.Page,
	})
}

func (h *AdminHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	admin, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Admin tidak ditemukan")
		return
	}
	utils.OK(w, "Fetch success", dto.NewAdminResponse(*admin))
}

// GET /api/admins/pic
func (h *AdminHandler) ListPIC(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListPIC(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data PIC")
		return
	}
	utils.OK(w, "Fetch success", dto.NewAdminResponseList(data))
}

// GET /api/admins/pic-client
func (h *AdminHandler) ListPICClient(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListPICClient(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal mengambil data PIC client")
		return
	}
	utils.OK(w, "Fetch success", dto.NewAdminResponseList(data))
}

// POST /api/admins
func (h *AdminHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAdminRequest
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

	id := utils.GenerateSequentialID("ADM")
	token := utils.RandomHex(20)

	if err := h.repo.Create(r.Context(), id, req.Email, req.Name, int(req.RegionID), req.RoleIDPtr(), req.DivisionIDPtr(), token, req.Pic, req.PicClient); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal membuat admin")
		return
	}

	activationURL := fmt.Sprintf("%s/activate?token=%s&email=%s", h.frontendURL, token, req.Email)
	_ = h.mail.SendActivationEmail(req.Email, req.Name, activationURL)

	notifyList, _ := h.repo.ListByAccessSlug(r.Context(), "mail_new_admin_register")
	if len(notifyList) > 0 {
		emails := make([]string, 0, len(notifyList))
		for _, a := range notifyList {
			emails = append(emails, a.Email)
		}
		_ = h.mail.SendNewAdminRegisteredNotice(emails, req.Name, req.Email)
	}

	utils.Created(w, "Admin berhasil dibuat, email aktivasi telah dikirim", map[string]any{"admin_id": id})
}

// PUT /api/admins/{id}
func (h *AdminHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateAdminRequest
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
		utils.Error(w, http.StatusNotFound, "Admin tidak ditemukan")
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

	if err := h.repo.Update(r.Context(), id, req.RoleIDPtr(), req.DivisionIDPtr(), req.Email, req.Name, req.Pic, req.PicClient, int(req.RegionID)); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui admin")
		return
	}
	utils.OK(w, "Admin berhasil diperbarui", nil)
}

// POST /api/admins/{id}/image  (multipart/form-data, field: image)
func (h *AdminHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Ukuran file maksimal 10MB")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "File gambar wajib diunggah")
		return
	}
	defer file.Close()

	allowedExt := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExt[ext] {
		utils.Error(w, http.StatusBadRequest, "Format gambar harus jpg, jpeg, png, atau gif")
		return
	}

	imgPath, thumbPath, err := h.saveAdminImage(file, ext)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan gambar")
		return
	}

	if err := h.repo.UpdateImages(r.Context(), id, imgPath, thumbPath); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui foto admin")
		return
	}

	utils.OK(w, "Foto berhasil diperbarui", map[string]any{"image": imgPath, "image_thumb": thumbPath})
}

func (h *AdminHandler) saveAdminImage(file multipart.File, ext string) (string, string, error) {
	if err := os.MkdirAll(filepath.Join(h.uploadDir, "admin"), 0o755); err != nil {
		return "", "", err
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	relPath := filepath.Join("uploads", "admin", filename)
	fullPath := filepath.Join(h.uploadDir, "admin", filename)

	out, err := os.Create(fullPath)
	if err != nil {
		return "", "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", "", err
	}

	return relPath, relPath, nil
}

// POST /api/admins/{id}/resend-activation
func (h *AdminHandler) ResendActivation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	admin, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "Admin tidak ditemukan")
		return
	}
	token := utils.RandomHex(20)
	if err := h.repo.UpdateToken(r.Context(), id, token); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui token aktivasi")
		return
	}
	activationURL := fmt.Sprintf("%s/activate?token=%s&email=%s", h.frontendURL, token, admin.Email)
	_ = h.mail.SendActivationEmail(admin.Email, admin.Name, activationURL)
	utils.OK(w, "Email aktivasi berhasil dikirim ulang", nil)
}

// POST /api/admins/{id}/deactivate
func (h *AdminHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	h.setActive(w, r, "inactive")
}

// POST /api/admins/{id}/activate
func (h *AdminHandler) ActivateExisting(w http.ResponseWriter, r *http.Request) {
	h.setActive(w, r, "active")
}

func (h *AdminHandler) setActive(w http.ResponseWriter, r *http.Request, status string) {
	id := chi.URLParam(r, "id")
	if err := h.repo.ChangeActive(r.Context(), id, status); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui status admin")
		return
	}
	utils.OK(w, "Status admin berhasil diperbarui", nil)
}

// DELETE /api/admins/{id}
func (h *AdminHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	actorID, _ := actorFromContext(r.Context())
	if actorID == id {
		utils.Error(w, http.StatusBadRequest, "Tidak dapat menghapus akun sendiri")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menghapus admin")
		return
	}
	utils.OK(w, "Admin berhasil dihapus", nil)
}