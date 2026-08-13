package middleware

// NOTE: package middleware meng-import "rms-backend/internal/utils", yang di
// jwt.go butuh golang-jwt/jwt/v5 dan google/uuid (dependency eksternal).
// Sandbox tempat file ini ditulis TIDAK punya akses ke proxy.golang.org untuk
// mengunduh dependency tsb, sehingga test ini belum sempat di-compile/dijalankan
// di lingkungan tersebut. Jalankan `go mod tidy && go test ./internal/middleware/...`
// di mesin Anda sendiri untuk memverifikasi.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"rms-backend/internal/utils"
)

// mockAccessChecker mengimplementasikan middleware.AccessChecker tanpa
// menyentuh database sungguhan - persis kontrak yang didefinisikan di access.go.
type mockAccessChecker struct {
	allowed map[string]bool // key: adminID+":"+slug
	err     error
}

func (m *mockAccessChecker) HasAccess(ctx context.Context, adminID string, slug string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.allowed[adminID+":"+slug], nil
}

// newRequestWithClaims menyisipkan *utils.Claims ASLI ke context (bukan mock
// struct) - WAJIB tipe konkret yang sama karena ClaimsFromContext melakukan
// type assertion langsung ke *utils.Claims, bukan sekadar duck-typing.
func newRequestWithClaims(adminID string) *http.Request {
	req := httptest.NewRequest("POST", "/api/po", nil)
	claims := &utils.Claims{AdminID: adminID}
	ctx := context.WithValue(req.Context(), ClaimsContextKey, claims)
	return req.WithContext(ctx)
}

func TestRequireAccess_AllowsWhenPermissionGranted(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"ADM001:edit_po": true}}
	handlerCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	mw := RequireAccess(checker, "edit_po")
	rr := httptest.NewRecorder()
	req := newRequestWithClaims("ADM001")

	mw(next).ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("expected next handler to be called when access is granted")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireAccess_BlocksWhenPermissionDenied(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{}}
	handlerCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	mw := RequireAccess(checker, "delete_po")
	rr := httptest.NewRecorder()
	req := newRequestWithClaims("ADM002")

	mw(next).ServeHTTP(rr, req)

	if handlerCalled {
		t.Error("expected next handler NOT to be called when access is denied")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestRequireAccess_BlocksWhenCheckerErrors(t *testing.T) {
	checker := &mockAccessChecker{err: errors.New("db connection lost")}
	handlerCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	mw := RequireAccess(checker, "edit_po")
	rr := httptest.NewRecorder()
	req := newRequestWithClaims("ADM001")

	mw(next).ServeHTTP(rr, req)

	if handlerCalled {
		t.Error("expected next handler NOT to be called when checker errors")
	}
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when access check itself fails, got %d", rr.Code)
	}
}

func TestRequireAccess_BlocksWhenNoClaims(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"ADM001:edit_po": true}}
	handlerCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	mw := RequireAccess(checker, "edit_po")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/po", nil) // TANPA claims di context

	mw(next).ServeHTTP(rr, req)

	if handlerCalled {
		t.Error("expected next handler NOT to be called without claims in context")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when claims are missing, got %d", rr.Code)
	}
}

func TestRequireAccess_DifferentSlugDenied(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"ADM001:edit_po": true}}

	mw := RequireAccess(checker, "delete_po")
	rr := httptest.NewRecorder()
	req := newRequestWithClaims("ADM001")

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for mismatched slug, got %d", rr.Code)
	}
}
