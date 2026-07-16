package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"rms-backend/internal/middleware"
)

func decodeJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
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
