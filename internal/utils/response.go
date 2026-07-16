package utils

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

func JSON(w http.ResponseWriter, status int, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: success,
		Message: message,
		Data:    data,
	})
}

// JSONMeta sama seperti JSON tapi menyertakan blok "meta" terpisah, dipakai
// untuk data yang dipaginasi (total_data, total_page, dsb).
func JSONMeta(w http.ResponseWriter, status int, success bool, message string, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: success,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func OK(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusOK, true, message, data)
}

func Created(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusCreated, true, message, data)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, false, message, nil)
}

// ClientIP mengambil IP asli client, memperhitungkan header dari reverse
// proxy (X-Forwarded-For / X-Real-IP). Pastikan hanya dipercaya jika app
// berjalan di belakang proxy/load balancer yang terpercaya.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// ambil IP pertama dalam daftar
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	host := r.RemoteAddr
	// buang port jika ada
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}
