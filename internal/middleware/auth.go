package middleware

import (
	"context"
	"net/http"
	"strings"

	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type ctxKey string

const ClaimsContextKey ctxKey = "auth_claims"

// Auth memverifikasi header "Authorization: Bearer <token>", memastikan
// token valid (signature + belum expired) dan belum di-blacklist (logout).
// Claims yang tervalidasi disisipkan ke request context untuk dipakai
// handler berikutnya (mis. mengambil admin_id aktor untuk logging).
func Auth(jwtManager *utils.JWTManager, authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				utils.Error(w, http.StatusUnauthorized, "Token tidak ditemukan")
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			claims, err := jwtManager.ParseAccessToken(tokenStr)
			if err != nil {
				utils.Error(w, http.StatusUnauthorized, "Token tidak valid atau sudah kedaluwarsa")
				return
			}

			if authService.IsAccessTokenBlacklisted(r.Context(), claims.ID) {
				utils.Error(w, http.StatusUnauthorized, "Token sudah tidak berlaku, silakan login ulang")
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext adalah helper yang bisa dipakai handler manapun untuk
// mengambil identitas admin yang sedang login.
func ClaimsFromContext(ctx context.Context) (*utils.Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*utils.Claims)
	return claims, ok
}
