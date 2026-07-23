package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"rms-backend/internal/middleware"
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