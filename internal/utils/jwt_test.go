package utils

// CATATAN: file ini butuh github.com/golang-jwt/jwt/v5 dan github.com/google/uuid
// (dependency eksternal). Sandbox tempat file ini dibuat TIDAK punya akses ke
// proxy.golang.org, sehingga test ini belum bisa dijalankan/di-compile di
// lingkungan tersebut. Jalankan `go test ./internal/utils/...` di mesin Anda
// sendiri (dengan akses internet normal) untuk memverifikasi.

import (
	"testing"
	"time"
)

func newTestJWTManager() *JWTManager {
	return NewJWTManager(
		"test-access-secret-please-change",
		"test-refresh-secret-please-change",
		15*time.Minute,
		7*24*time.Hour,
		"rms-backend-test",
	)
}

func TestGenerateAccessToken_ReturnsNonEmptyTokenAndJTI(t *testing.T) {
	m := newTestJWTManager()
	roleID := 1
	token, jti, err := m.GenerateAccessToken("ADM001", "admin@rms.local", &roleID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	if jti == "" {
		t.Error("expected non-empty jti")
	}
}

func TestParseAccessToken_RoundTrip(t *testing.T) {
	m := newTestJWTManager()
	roleID := 2
	token, jti, err := m.GenerateAccessToken("ADM002", "staff@rms.local", &roleID)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	claims, err := m.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("unexpected error parsing token: %v", err)
	}
	if claims.AdminID != "ADM002" {
		t.Errorf("expected AdminID=ADM002, got %s", claims.AdminID)
	}
	if claims.Email != "staff@rms.local" {
		t.Errorf("expected Email=staff@rms.local, got %s", claims.Email)
	}
	if claims.RoleID == nil || *claims.RoleID != 2 {
		t.Errorf("expected RoleID=2, got %v", claims.RoleID)
	}
	if claims.Type != "access" {
		t.Errorf("expected Type=access, got %s", claims.Type)
	}
	if claims.ID != jti {
		t.Errorf("expected claims.ID to match returned jti %s, got %s", jti, claims.ID)
	}
}

func TestParseAccessToken_NilRoleID(t *testing.T) {
	m := newTestJWTManager()
	token, _, err := m.GenerateAccessToken("ADM003", "norole@rms.local", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	claims, err := m.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.RoleID != nil {
		t.Errorf("expected nil RoleID, got %v", *claims.RoleID)
	}
}

func TestParseAccessToken_RejectsRefreshToken(t *testing.T) {
	// Refresh token TIDAK boleh diterima sebagai access token, walau
	// signature-nya valid - Type harus dicocokkan secara eksplisit.
	m := newTestJWTManager()
	refreshToken, _, err := m.GenerateRefreshToken("ADM001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := m.ParseAccessToken(refreshToken); err == nil {
		t.Error("expected error when parsing a refresh token as access token")
	}
}

func TestParseRefreshToken_RejectsAccessToken(t *testing.T) {
	m := newTestJWTManager()
	roleID := 1
	accessToken, _, err := m.GenerateAccessToken("ADM001", "admin@rms.local", &roleID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := m.ParseRefreshToken(accessToken); err == nil {
		t.Error("expected error when parsing an access token as refresh token")
	}
}

func TestParseAccessToken_RejectsWrongSecret(t *testing.T) {
	m1 := NewJWTManager("secret-a", "refresh-a", 15*time.Minute, time.Hour, "issuer-a")
	m2 := NewJWTManager("secret-b", "refresh-b", 15*time.Minute, time.Hour, "issuer-b")

	roleID := 1
	token, _, err := m1.GenerateAccessToken("ADM001", "admin@rms.local", &roleID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Token yang ditandatangani manager A tidak boleh valid di manager B
	// (secret berbeda) - mensimulasikan token dipalsukan/rotated secret.
	if _, err := m2.ParseAccessToken(token); err == nil {
		t.Error("expected error when parsing token signed with a different secret")
	}
}

func TestParseAccessToken_RejectsGarbage(t *testing.T) {
	m := newTestJWTManager()
	if _, err := m.ParseAccessToken("not.a.valid.jwt.token"); err == nil {
		t.Error("expected error when parsing a malformed token string")
	}
}

func TestParseAccessToken_RejectsExpiredToken(t *testing.T) {
	// TTL sangat pendek (1 nanodetik) supaya token langsung expired begitu
	// diparse - menghindari test yang butuh sleep sungguhan/flaky.
	m := NewJWTManager("secret", "refresh-secret", 1*time.Nanosecond, time.Hour, "issuer")
	roleID := 1
	token, _, err := m.GenerateAccessToken("ADM001", "admin@rms.local", &roleID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	time.Sleep(2 * time.Millisecond) // pastikan waktu sudah lewat TTL

	if _, err := m.ParseAccessToken(token); err == nil {
		t.Error("expected error when parsing an expired token")
	}
}

func TestGenerateRefreshToken_DifferentJTIEachCall(t *testing.T) {
	m := newTestJWTManager()
	_, jti1, err := m.GenerateRefreshToken("ADM001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, jti2, err := m.GenerateRefreshToken("ADM001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jti1 == jti2 {
		t.Error("expected different jti for each refresh token generation")
	}
}

func TestAccessTTL_RefreshTTL_Exposed(t *testing.T) {
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour
	m := NewJWTManager("a", "b", accessTTL, refreshTTL, "issuer")
	if m.AccessTTL() != accessTTL {
		t.Errorf("expected AccessTTL()=%v, got %v", accessTTL, m.AccessTTL())
	}
	if m.RefreshTTL() != refreshTTL {
		t.Errorf("expected RefreshTTL()=%v, got %v", refreshTTL, m.RefreshTTL())
	}
}
