package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"rms-backend/internal/middleware"
	"rms-backend/internal/utils"
)

const maxJSONBodyBytes = 1 << 20 // 1 MB

func decodeJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(nil, r.Body, maxJSONBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// actorFromContext mengambil admin_id & nama dari JWT claims - dipakai untuk
// mengisi "siapa yang melakukan aksi ini" pada pencatatan T_Activity, dsb.
// Menggantikan $_SESSION['admin_id'] / $_SESSION['admin_username'] di PHP.
func actorFromContext(ctx context.Context) (adminID string, ok bool) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.AdminID, true
}

type prVisibility interface {
	IsAdminRelated(ctx context.Context, prID int, adminID string) (bool, error)
}

// requirePRVisible menulis error dan mengembalikan false bila aktor tidak boleh melihat PR ini.
func requirePRVisible(w http.ResponseWriter, r *http.Request, v prVisibility, prID int) bool {
	adminID, ok := actorFromContext(r.Context())
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
		return false
	}
	visible, err := v.IsAdminRelated(r.Context(), prID, adminID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa akses ke purchase request")
		return false
	}
	if !visible {
		utils.Error(w, http.StatusForbidden, "Anda tidak memiliki akses ke purchase request ini")
		return false
	}
	return true
}