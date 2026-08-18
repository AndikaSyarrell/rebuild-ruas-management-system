package middleware

import (
	"net/http"
	"time"

	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// RequestLogger mencatat SETIAP request (method, path, status, durasi, IP,
// admin_id jika sudah lolos Auth) lewat service.Logger - jadi semua fitur
// otomatis punya audit trail dasar tanpa perlu logger.Log(...) manual di
// setiap handler. WAJIB didaftarkan SETELAH middleware Auth supaya admin_id
// ikut tercatat (lihat instruksi wiring di bawah).
func RequestLogger(logger *service.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			status := service.LogStatusSuccess
			switch {
			case rec.status >= 500:
				status = service.LogStatusError
			case rec.status >= 400:
				status = service.LogStatusWarning
			}

			var userID *string
			if claims, ok := ClaimsFromContext(r.Context()); ok {
				userID = &claims.AdminID
			}

			logger.Log(service.LogEntry{
				UserID:    userID,
				Module:    "http",
				Action:    r.Method + " " + r.URL.Path,
				Status:    status,
				Message:   "Request selesai diproses",
				IP:        utils.ClientIP(r),
				UserAgent: r.UserAgent(),
				Metadata: map[string]any{
					"status_code": rec.status,
					"duration_ms": time.Since(start).Milliseconds(),
					"query":       r.URL.RawQuery,
				},
			})
		})
	}
}