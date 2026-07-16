package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"rms-backend/internal/middleware"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type AuthHandler struct {
	authService      *service.AuthService
	throttle         *middleware.LoginThrottle
	pwChangeThrottle *middleware.PasswordChangeThrottle
	mail             *service.MailService
	logger           *service.Logger
	frontendBaseURL  string
}

func NewAuthHandler(authService *service.AuthService, throttle *middleware.LoginThrottle, pwChangeThrottle *middleware.PasswordChangeThrottle, mail *service.MailService, logger *service.Logger, frontendBaseURL string) *AuthHandler {
	return &AuthHandler{authService: authService, throttle: throttle, pwChangeThrottle: pwChangeThrottle, mail: mail, logger: logger, frontendBaseURL: frontendBaseURL}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type resetPasswordRequest struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetWithCodeRequest struct {
	ResetCode   string `json:"reset_code"`
	NewPassword string `json:"new_password"`
}

type activateRequest struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

// Login godoc
// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ip := utils.ClientIP(r)
	userAgent := r.UserAgent()

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || req.Password == "" {
		utils.Error(w, http.StatusBadRequest, "Email dan password wajib diisi")
		return
	}

	// --- Throttling: cek SEBELUM menyentuh database / bcrypt ---
	allowed, reason, retryAfter := h.throttle.PreCheck(ctx, ip, req.Email)
	if !allowed {
		h.logger.Log(service.LogEntry{
			Module:    "auth",
			Action:    "login_blocked",
			Status:    service.LogStatusWarning,
			Message:   "Login diblokir oleh throttling: " + reason,
			IP:        ip,
			UserAgent: userAgent,
			Metadata:  map[string]any{"email": req.Email, "reason": reason, "retry_after_seconds": int(retryAfter.Seconds())},
		})

		msg := "Terlalu banyak percobaan login, silakan coba lagi nanti"
		if reason == "account_locked" {
			msg = "Akun sementara dikunci karena terlalu banyak percobaan gagal. Coba lagi nanti."
		}
		w.Header().Set("Retry-After", retryAfter.String())
		utils.Error(w, http.StatusTooManyRequests, msg)
		return
	}

	pair, admin, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		h.throttle.RegisterFailure(ctx, req.Email)

		message := "Login gagal"
		clientMsg := "Email atau password salah"

		if errors.Is(err, service.ErrAccountInactive) {
			message = "Login gagal: akun nonaktif"
			clientMsg = "Akun tidak aktif, hubungi administrator"
		}

		h.logger.Log(service.LogEntry{
			Module: "auth", Action: "login_failed", Status: service.LogStatusWarning, Message: message,
			IP: ip, UserAgent: userAgent,
			Metadata: map[string]any{"email": req.Email},
		})

		utils.Error(w, http.StatusUnauthorized, clientMsg)
		return
	}

	h.throttle.RegisterSuccess(ctx, req.Email)

	h.logger.Log(service.LogEntry{
		UserID: &admin.ID, Module: "auth", Action: "login_success",
		Status: service.LogStatusSuccess, Message: "Login berhasil",
		IP: ip, UserAgent: userAgent,
	})

	utils.OK(w, "Login berhasil", map[string]any{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    pair.ExpiresIn,
		"admin":         admin.ToPublic(),
	})
}

// Refresh godoc
// POST /api/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ip := utils.ClientIP(r)

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.Error(w, http.StatusBadRequest, "refresh_token wajib diisi")
		return
	}

	pair, err := h.authService.Refresh(ctx, req.RefreshToken)
	if err != nil {
		h.logger.Log(service.LogEntry{
			Module: "auth", Action: "token_refresh_failed", Status: service.LogStatusWarning,
			Message: err.Error(), IP: ip,
		})
		utils.Error(w, http.StatusUnauthorized, "Refresh token tidak valid, silakan login ulang")
		return
	}

	h.logger.Log(service.LogEntry{
		Module: "auth", Action: "token_refresh_success", Status: service.LogStatusSuccess, IP: ip,
	})

	utils.OK(w, "Token berhasil diperbarui", map[string]any{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    pair.ExpiresIn,
	})
}

// Logout godoc
// POST /api/auth/logout  (protected, butuh Authorization header)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	var req refreshRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // refresh_token opsional saat logout

	if err := h.authService.Logout(ctx, claims, req.RefreshToken); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal logout")
		return
	}

	h.logger.Log(service.LogEntry{
		UserID: &claims.AdminID, Module: "auth", Action: "logout",
		Status: service.LogStatusInfo, Message: "Logout berhasil", IP: utils.ClientIP(r),
	})

	utils.OK(w, "Logout berhasil", nil)
}

// ResetPassword godoc
// POST /api/auth/reset-password  (protected, butuh Authorization header)
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ip := utils.ClientIP(r)

	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}

	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
		utils.Error(w, http.StatusBadRequest, "Semua field wajib diisi")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		utils.Error(w, http.StatusBadRequest, "Konfirmasi password baru tidak cocok")
		return
	}
	if err := utils.ValidatePasswordStrength(req.NewPassword); err != nil {
		utils.Error(w, http.StatusBadRequest, "Password baru minimal 8 karakter dan mengandung huruf besar, huruf kecil, serta angka")
		return
	}

	allowed, retryAfter := h.pwChangeThrottle.PreCheck(ctx, claims.AdminID)
	if !allowed {
		w.Header().Set("Retry-After", retryAfter.String())
		utils.Error(w, http.StatusTooManyRequests, "Terlalu banyak percobaan gagal, silakan coba lagi nanti")
		return
	}

	err := h.authService.ChangePassword(ctx, claims, req.OldPassword, req.NewPassword)
	if err != nil {
		httpStatus := http.StatusBadRequest
		clientMsg := "Gagal mengganti password"

		switch {
		case errors.Is(err, service.ErrWrongOldPassword):
			h.pwChangeThrottle.RegisterFailure(ctx, claims.AdminID)
			clientMsg = "Password lama salah"
			httpStatus = http.StatusUnauthorized
		case errors.Is(err, service.ErrSamePassword):
			clientMsg = "Password baru tidak boleh sama dengan password lama"
		case errors.Is(err, service.ErrAccountInactive):
			clientMsg = "Akun tidak aktif"
			httpStatus = http.StatusForbidden
		default:
			httpStatus = http.StatusInternalServerError
			clientMsg = "Terjadi kesalahan pada server"
		}

		h.logger.Log(service.LogEntry{
			UserID: &claims.AdminID, Module: "auth", Action: "password_change_failed",
			Status: service.LogStatusWarning, Message: err.Error(), IP: ip,
		})

		utils.Error(w, httpStatus, clientMsg)
		return
	}

	h.pwChangeThrottle.RegisterSuccess(ctx, claims.AdminID)

	h.logger.Log(service.LogEntry{
		UserID: &claims.AdminID, Module: "auth", Action: "password_change_success",
		Status: service.LogStatusSuccess, Message: "Password berhasil diganti, semua sesi lain di-revoke",
		IP: ip,
	})

	utils.OK(w, "Password berhasil diganti. Silakan login ulang.", nil)
}

// ForgotPassword godoc
// POST /api/auth/forgot-password
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		utils.Error(w, http.StatusBadRequest, "Email wajib diisi")
		return
	}

	code, err := h.authService.RequestPasswordReset(r.Context(), req.Email)
	// Selalu balas sukses agar endpoint ini tidak bisa dipakai enumerasi email terdaftar.
	if err == nil {
		resetURL := h.frontendBaseURL + "/reset-password?code=" + code
		_ = h.mail.SendForgetPasswordEmail(req.Email, resetURL)
	}
	utils.OK(w, "Jika email terdaftar, tautan reset password telah dikirim.", nil)
}

// ResetPasswordWithCode godoc
// POST /api/auth/reset-password-code
func (h *AuthHandler) ResetPasswordWithCode(w http.ResponseWriter, r *http.Request) {
	var req resetWithCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ResetCode == "" || req.NewPassword == "" {
		utils.Error(w, http.StatusBadRequest, "Data tidak lengkap")
		return
	}
	if err := h.authService.ResetPasswordWithCode(r.Context(), req.ResetCode, req.NewPassword); err != nil {
		utils.Error(w, http.StatusBadRequest, "Kode reset tidak valid atau password lemah")
		return
	}
	utils.OK(w, "Password berhasil direset, silakan login.", nil)
}

// Activate godoc
// POST /api/auth/activate
func (h *AuthHandler) Activate(w http.ResponseWriter, r *http.Request) {
	var req activateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Token == "" || req.Password == "" {
		utils.Error(w, http.StatusBadRequest, "Data tidak lengkap")
		return
	}
	if err := h.authService.ActivateAccount(r.Context(), req.Email, req.Token, req.Password); err != nil {
		utils.Error(w, http.StatusBadRequest, "Aktivasi gagal, token tidak valid atau password terlalu lemah")
		return
	}
	utils.OK(w, "Akun berhasil diaktifkan, silakan login.", nil)
}

// Me godoc
// GET /api/auth/me  (protected)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return
	}
	utils.OK(w, "OK", map[string]any{
		"admin_id": claims.AdminID,
		"email":    claims.Email,
		"role_id":  claims.RoleID,
	})
}
