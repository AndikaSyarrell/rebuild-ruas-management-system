package handlers

import (
	"errors"
	"net/http"
	"fmt"
	"database/sql"
	"golang.org/x/crypto/bcrypt"
	"context"

	"rms-backend/internal/config"
	"rms-backend/internal/dto"
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
	cfg              *config.Config
}

func NewAuthHandler(authService *service.AuthService, throttle *middleware.LoginThrottle, pwChangeThrottle *middleware.PasswordChangeThrottle, mail *service.MailService, logger *service.Logger, frontendBaseURL string, cfg *config.Config) *AuthHandler {
	return &AuthHandler{authService: authService, throttle: throttle, pwChangeThrottle: pwChangeThrottle, mail: mail, logger: logger, frontendBaseURL: frontendBaseURL, cfg: cfg}
}

// setRefreshCookie menuliskan refresh_token sebagai httpOnly cookie sesuai
// BACKEND_AUTH_README.md - token tidak pernah keluar lewat body JSON.
func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.RefreshCookieName,
		Value:    token,
		Path:     h.cfg.RefreshCookiePath,
		Domain:   h.cfg.RefreshCookieDomain,
		HttpOnly: true,
		Secure:   h.cfg.RefreshCookieSecure,
		SameSite: sameSiteFromString(h.cfg.RefreshCookieSameSite),
		MaxAge:   int(h.authService.RefreshTTL().Seconds()),
	})
}

// clearRefreshCookie menghapus cookie refresh_token (dipakai saat logout /
// saat refresh gagal, supaya browser tidak terus mengirim token yang sudah invalid).
func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.RefreshCookieName,
		Value:    "",
		Path:     h.cfg.RefreshCookiePath,
		Domain:   h.cfg.RefreshCookieDomain,
		HttpOnly: true,
		Secure:   h.cfg.RefreshCookieSecure,
		SameSite: sameSiteFromString(h.cfg.RefreshCookieSameSite),
		MaxAge:   -1,
	})
}

func sameSiteFromString(v string) http.SameSite {
	switch v {
	case "Lax":
		return http.SameSiteLaxMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

// Login godoc
// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ip := utils.ClientIP(r)
	userAgent := r.UserAgent()

	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
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

		// --- DEBUG: log the SPECIFIC underlying reason internally ---
		// Client always sees the generic message above (don't change that —
		// leaking "user not found" vs "wrong password" is a security smell).
		// But we need to know which branch actually triggered, so log it.
		debugReason := "unknown"
		switch {
		case errors.Is(err, service.ErrAccountInactive):
			message = "Login gagal: akun nonaktif"
			clientMsg = "Akun tidak aktif, hubungi administrator"
			debugReason = "account_inactive"
		case errors.Is(err, sql.ErrNoRows):
			debugReason = "user_not_found_in_db"
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			debugReason = "password_hash_mismatch"
		case errors.Is(err, context.DeadlineExceeded):
			debugReason = "db_query_timeout"
		default:
			// This is the one to watch for right now — if it hits "default",
			// authService.Login is returning an error type we don't recognize
			// here, which means it's NOT sql.ErrNoRows or bcrypt mismatch.
			// That points at a query/scan bug, not a credentials bug.
			debugReason = fmt.Sprintf("unclassified: %v", err)
		}

		h.logger.Log(service.LogEntry{
			Module: "auth", Action: "login_failed", Status: service.LogStatusWarning, Message: message,
			IP: ip, UserAgent: userAgent,
			Metadata: map[string]any{
				"email":        req.Email,
				"debug_reason": debugReason,
				"raw_error":    err.Error(), // remove once bug is found — don't leave raw errors in logs long-term
			},
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

	// refresh_token TIDAK dimasukkan ke body JSON - hanya keluar lewat
	// Set-Cookie httpOnly (lihat BACKEND_AUTH_README.md).
	h.setRefreshCookie(w, pair.RefreshToken)

	resp := dto.LoginResponse{
		TokenPairResponse: dto.TokenPairResponse{
			AccessToken: pair.AccessToken,
			TokenType:   "Bearer",
			ExpiresIn:   pair.ExpiresIn,
		},
		Admin: dto.NewAdminResponse(*admin),
	}
	utils.OK(w, "Login berhasil", resp)
}

// Refresh godoc
// POST /api/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ip := utils.ClientIP(r)

	// refresh_token sekarang dibaca dari httpOnly cookie, bukan body JSON
	// (lihat BACKEND_AUTH_README.md - endpoint refresh tidak butuh body).
	cookie, err := r.Cookie(h.cfg.RefreshCookieName)
	if err != nil || cookie.Value == "" {
		utils.Error(w, http.StatusUnauthorized, "Sesi tidak ditemukan, silakan login ulang")
		return
	}

	pair, err := h.authService.Refresh(ctx, cookie.Value)
	if err != nil {
		h.logger.Log(service.LogEntry{
			Module: "auth", Action: "token_refresh_failed", Status: service.LogStatusWarning,
			Message: err.Error(), IP: ip,
		})
		// Cookie yang sudah tidak valid (habis rotasi/reuse/expired) dibersihkan
		// supaya browser tidak terus mengirim token mati.
		h.clearRefreshCookie(w)
		utils.Error(w, http.StatusUnauthorized, "Refresh token tidak valid, silakan login ulang")
		return
	}

	// Rotasi: cookie lama diganti dengan refresh_token baru.
	h.setRefreshCookie(w, pair.RefreshToken)

	h.logger.Log(service.LogEntry{
		Module: "auth", Action: "token_refresh_success", Status: service.LogStatusSuccess, IP: ip,
	})

	utils.OK(w, "Token berhasil diperbarui", dto.TokenPairResponse{
		AccessToken: pair.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   pair.ExpiresIn,
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

	// refresh_token dibaca dari cookie (opsional - endpoint tetap boleh
	// dipanggil tanpa cookie, misalnya cookie sudah kedaluwarsa lebih dulu).
	refreshToken := ""
	if cookie, err := r.Cookie(h.cfg.RefreshCookieName); err == nil {
		refreshToken = cookie.Value
	}

	if err := h.authService.Logout(ctx, claims, refreshToken); err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal logout")
		return
	}

	h.clearRefreshCookie(w)

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

	var req dto.ResetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
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
	var req dto.ForgotPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
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
	var req dto.ResetWithCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
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
	var req dto.ActivateRequest
	if err := decodeJSON(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Body permintaan tidak valid")
		return
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
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
	utils.OK(w, "OK", dto.MeResponse{
		AdminID: claims.AdminID,
		Email:   claims.Email,
		RoleID:  claims.RoleID,
	})
}