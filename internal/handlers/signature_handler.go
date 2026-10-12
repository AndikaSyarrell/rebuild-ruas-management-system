package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"database/sql"
	"errors"
	"mime"

	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

type SignatureHandler struct {
	repo      *repository.AdminSignatureRepo
	uploadDir string
}

func NewSignatureHandler(repo *repository.AdminSignatureRepo, uploadDir string) *SignatureHandler {
	return &SignatureHandler{repo: repo, uploadDir: uploadDir}
}

var allowedSignatureMIME = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
}
var allowedSignatureExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true}

// GET /api/admins/me/signature
func (h *SignatureHandler) Mine(w http.ResponseWriter, r *http.Request) {
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	sig, err := h.repo.GetLatestByAdmin(r.Context(), adminID)
		if errors.Is(err, sql.ErrNoRows) {
			utils.Error(w, http.StatusNotFound, "Anda belum memiliki tanda tangan terdaftar")
			return
		}
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal mengambil tanda tangan")
			return
		}
	utils.OK(w, "Fetch success", map[string]any{
		"signature_id": sig.ID, "signature_file": sig.File, "signature_name_pic": sig.NamePic,
	})
}

func (h *SignatureHandler) File(w http.ResponseWriter, r *http.Request){
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "tidak terauntentikasi")
		return
	}

	sig, err := h.repo.GetLatestByAdmin(r.Context(), adminID)
	if errors.Is(err, sql.ErrNoRows){
		utils.Error(w, http.StatusNotFound, "tanda tangan tidak ditemukan")
		return
	}
	if err != nil{
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil data, kesalahan server")
		return
	}
	
	f, err := os.Open(filepath.Join(h.uploadDir, "..", sig.File))
	if err != nil{
		utils.Error(w, http.StatusInternalServerError, "tanda tangan tidak ditemukan diserver")
		return
	}

	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.Size() == 0 {
		utils.Error(w, http.StatusInternalServerError, "file tanda tangan tidak ditemukan atau rusak")
		return
	}

	ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(sig.File)))
	if ct == "" {
		ct = "imge/png"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, "", fi.ModTime(), f)
}

// POST /api/admins/me/signature (multipart/form-data: file, name_pic)
func (h *SignatureHandler) Upload(w http.ResponseWriter, r *http.Request) {
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "Ukuran file maksimal 5MB")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "File tanda tangan wajib diunggah")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedSignatureExt[ext] {
		utils.Error(w, http.StatusBadRequest, "Format tanda tangan harus png, jpg, atau jpeg")
		return
	}
	if err := utils.ValidateFileContent(file, allowedSignatureMIME); err != nil {
		utils.Error(w, http.StatusBadRequest, "Isi file tidak sesuai dengan format gambar yang diklaim")
		return
	}

	namePic := r.FormValue("name_pic")
	if namePic == "" {
		utils.Error(w, http.StatusBadRequest, "Nama PIC wajib diisi")
		return
	}

	if err := os.MkdirAll(filepath.Join(h.uploadDir, "signature"), 0o755); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyiapkan penyimpanan")
		return
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	relPath := filepath.Join("uploads", "signature", filename)
	fullPath := filepath.Join(h.uploadDir, "signature", filename)

	out, err := os.Create(fullPath)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}

	id, err := h.repo.Create(r.Context(), adminID, namePic, relPath)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan tanda tangan")
		return
	}
	utils.Created(w, "Tanda tangan berhasil diunggah", map[string]any{"signature_id": id, "file": relPath})
}