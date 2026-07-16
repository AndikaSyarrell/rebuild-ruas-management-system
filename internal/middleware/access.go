package middleware

import (
	"context"
	"net/http"

	"rms-backend/internal/utils"
)

// AccessChecker adalah kontrak minimal yang dibutuhkan middleware RBAC ini.
// Diimplementasikan oleh repository.AccessRepo (lihat internal/repository/access_repo.go),
// namun didefinisikan di sini agar middleware tidak perlu import package repository secara langsung.
type AccessChecker interface {
	HasAccess(ctx context.Context, adminID string, slug string) (bool, error)
}

// RequireAccess adalah pengganti pola berulang di PHP:
//
//	$acsp_edit = $obj_access->check_access($obj_cruds, $_SESSION['admin_id'], 'edit_po');
//	if ($acsp_edit == 0) { header("Location:".$path['dashboard']); }
//
// Middleware ini WAJIB dipasang SETELAH Auth() karena bergantung pada claims di context.
// slug merujuk ke access_slug pada tabel T_Access (mis. "edit_po", "create_client", dst).
func RequireAccess(checker AccessChecker, slug string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				utils.Error(w, http.StatusUnauthorized, "Tidak terautentikasi")
				return
			}

			allowed, err := checker.HasAccess(r.Context(), claims.AdminID, slug)
			if err != nil {
				utils.Error(w, http.StatusInternalServerError, "Gagal memeriksa hak akses")
				return
			}
			if !allowed {
				utils.Error(w, http.StatusForbidden, "Anda tidak memiliki akses untuk aksi ini")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
