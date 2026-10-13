package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
		ct = "image/png"
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

	// 1 Akun = 1 Signature File:
	// Nama file distandarkan berdasarkan ID akun admin sehingga file baru otomatis
	// menggantikan (overwrite) file lama milik akun yang sama di folder storage.
	safeAdminID := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, adminID)

	filename := fmt.Sprintf("sig_%s%s", safeAdminID, ext)
	relPath := filepath.Join("uploads", "signature", filename)
	fullPath := filepath.Join(h.uploadDir, "signature", filename)

	if err := os.MkdirAll(filepath.Join(h.uploadDir, "signature"), 0o755); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal menyiapkan penyimpanan")
		return
	}

	// Ambil semua riwayat tanda tangan lama milik admin ini dari DB
	oldSigs, _ := h.repo.ListByAdmin(r.Context(), adminID)

	// Bersihkan file-file lama dari storage jika path-nya bukan file yang baru ini
	// (misalnya file lama yang masih menggunakan format timestamp acak)
	for _, old := range oldSigs {
		if old.File != relPath {
			cleanOld := strings.TrimLeft(filepath.Clean(old.File), `/\`)
			_ = os.Remove(filepath.Join(h.uploadDir, "..", cleanOld))
			_ = os.Remove(filepath.Join(h.uploadDir, "signature", filepath.Base(old.File)))
		}
	}

	// Hapus juga file dengan format ekstensi lain milik akun ini jika ada (misal sebelumnya .jpg, sekarang .png)
	for otherExt := range allowedSignatureExt {
		if otherExt != ext {
			_ = os.Remove(filepath.Join(h.uploadDir, "signature", fmt.Sprintf("sig_%s%s", safeAdminID, otherExt)))
		}
	}

	// Tulis/timpa (overwrite) file signature baru ke storage
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

	// Perbarui database: jika sudah ada record untuk admin ini, lakukan UPDATE.
	// Jika belum pernah ada, buat record baru dengan INSERT.
	var sigID int64
	if len(oldSigs) > 0 {
		primary := oldSigs[0]
		sigID = int64(primary.ID)
		if err := h.repo.Update(r.Context(), primary.ID, namePic, relPath); err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal memperbarui tanda tangan di database")
			return
		}
		// Hapus sisa duplikat record jika ada dari pengujian sebelumnya
		for _, dup := range oldSigs[1:] {
			_ = h.repo.Delete(r.Context(), dup.ID)
		}
	} else {
		newID, err := h.repo.Create(r.Context(), adminID, namePic, relPath)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "Gagal menyimpan tanda tangan")
			return
		}
		sigID = newID
	}

	utils.Created(w, "Tanda tangan berhasil disimpan", map[string]any{"signature_id": sigID, "file": relPath})
}