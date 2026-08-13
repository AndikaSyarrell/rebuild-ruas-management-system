package service

// NOTE: butuh github.com/DATA-DOG/go-sqlmock, github.com/alicebob/miniredis/v2,
// dan golang.org/x/crypto/bcrypt (semua dependency eksternal). Sandbox tempat
// file ini ditulis tidak punya akses ke proxy.golang.org untuk mengunduhnya,
// sehingga belum sempat di-compile/dijalankan di lingkungan tersebut.
// Jalankan `go test ./internal/service/...` di mesin Anda sendiri (dengan
// akses internet normal) untuk memverifikasi.

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

// adminSelectColumns HARUS persis sama urutannya dengan adminSelectWithJoins
// di admin_repo.go, supaya sqlmock.NewRows menghasilkan baris yang benar-benar
// bisa di-scan tanpa "sql: Scan error" akibat mismatch jumlah/urutan kolom.
var adminSelectColumns = []string{
	"admin_id", "admin_ref_region", "admin_ref_role", "admin_token", "admin_reset_code",
	"admin_email", "admin_name", "admin_password", "admin_active", "admin_pic", "admin_pic_client",
	"admin_img", "admin_img_thmb", "admin_create_date", "admin_modify_date",
	"role_title", "role_slug", "region_title",
}

func newTestAuthService(t *testing.T) (*AuthService, sqlmock.Sqlmock, *miniredis.Miniredis, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	adminRepo := repository.NewAdminRepo(db)
	jwtManager := utils.NewJWTManager("access-secret", "refresh-secret", 15*time.Minute, 7*24*time.Hour, "rms-test")

	svc := NewAuthService(adminRepo, jwtManager, rdb)
	cleanup := func() {
		db.Close()
		rdb.Close()
		mr.Close()
	}
	return svc, mock, mr, cleanup
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, mock, _, cleanup := newTestAuthService(t)
	defer cleanup()

	hash, err := utils.HashPassword("Admin123!")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE a.admin_email = ? LIMIT 1")).
		WithArgs("admin@rms.local").
		WillReturnRows(sqlmock.NewRows(adminSelectColumns).AddRow(
			"ADM001", 1, 1, nil, nil,
			"admin@rms.local", "Super Admin", hash, "active", "yes", "no",
			nil, nil, now, now,
			"Super Admin", "super_admin", "Jakarta",
		))

	pair, admin, err := svc.Login(context.Background(), "admin@rms.local", "Admin123!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if admin.ID != "ADM001" {
		t.Errorf("expected admin ID=ADM001, got %s", admin.ID)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Error("expected non-empty access & refresh tokens")
	}
	if pair.ExpiresIn <= 0 {
		t.Error("expected positive expires_in")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc, mock, _, cleanup := newTestAuthService(t)
	defer cleanup()

	hash, _ := utils.HashPassword("Admin123!")
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE a.admin_email = ? LIMIT 1")).
		WithArgs("admin@rms.local").
		WillReturnRows(sqlmock.NewRows(adminSelectColumns).AddRow(
			"ADM001", 1, 1, nil, nil,
			"admin@rms.local", "Super Admin", hash, "active", "yes", "no",
			nil, nil, now, now,
			"Super Admin", "super_admin", "Jakarta",
		))

	_, _, err := svc.Login(context.Background(), "admin@rms.local", "WrongPassword!")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_InactiveAccount(t *testing.T) {
	svc, mock, _, cleanup := newTestAuthService(t)
	defer cleanup()

	hash, _ := utils.HashPassword("Admin123!")
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE a.admin_email = ? LIMIT 1")).
		WithArgs("pending@rms.local").
		WillReturnRows(sqlmock.NewRows(adminSelectColumns).AddRow(
			"ADM002", 1, 2, nil, nil,
			"pending@rms.local", "Pending Admin", hash, "inactive", "no", "no",
			nil, nil, now, now,
			"Staff", "staff", "Jakarta",
		))

	_, _, err := svc.Login(context.Background(), "pending@rms.local", "Admin123!")
	if !errors.Is(err, ErrAccountInactive) {
		t.Errorf("expected ErrAccountInactive, got %v", err)
	}
}

func TestAuthService_Login_EmailNotFound(t *testing.T) {
	svc, mock, _, cleanup := newTestAuthService(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE a.admin_email = ? LIMIT 1")).
		WithArgs("ghost@rms.local").
		WillReturnError(sql.ErrNoRows)

	_, _, err := svc.Login(context.Background(), "ghost@rms.local", "Whatever123!")
	if !errors.Is(err, ErrInvalidCredentials) {
		// Sengaja pesan generik "email atau password salah" untuk KEDUA kasus
		// (email tidak ada ATAUPUN password salah) - mencegah enumerasi
		// email terdaftar dari respons yang berbeda-beda.
		t.Errorf("expected ErrInvalidCredentials (generic message) for unknown email, got %v", err)
	}
}

func TestAuthService_Login_NilPasswordNeverMatches(t *testing.T) {
	// Admin yang belum pernah set password (mis. baru dibuat, belum aktivasi)
	// - admin_password NULL di DB - login harus SELALU ditolak, bukan panic.
	svc, mock, _, cleanup := newTestAuthService(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("WHERE a.admin_email = ? LIMIT 1")).
		WithArgs("newadmin@rms.local").
		WillReturnRows(sqlmock.NewRows(adminSelectColumns).AddRow(
			"ADM003", 1, 2, nil, nil,
			"newadmin@rms.local", "New Admin", nil, "inactive", "no", "no",
			nil, nil, now, now,
			"Staff", "staff", "Jakarta",
		))

	_, _, err := svc.Login(context.Background(), "newadmin@rms.local", "anything")
	if err == nil {
		t.Error("expected login to fail when admin_password is NULL")
	}
}

func TestAuthService_Refresh_RotatesToken(t *testing.T) {
	svc, mock, _, cleanup := newTestAuthService(t)
	defer cleanup()

	hash, _ := utils.HashPassword("Admin123!")
	now := time.Now()

	// Login dulu untuk dapat refresh token yang sah & tersimpan di Redis.
	mock.ExpectQuery(regexp.QuoteMeta("WHERE a.admin_email = ? LIMIT 1")).
		WithArgs("admin@rms.local").
		WillReturnRows(sqlmock.NewRows(adminSelectColumns).AddRow(
			"ADM001", 1, 1, nil, nil,
			"admin@rms.local", "Super Admin", hash, "active", "yes", "no",
			nil, nil, now, now,
			"Super Admin", "super_admin", "Jakarta",
		))
	pair, _, err := svc.Login(context.Background(), "admin@rms.local", "Admin123!")
	if err != nil {
		t.Fatalf("setup login failed: %v", err)
	}

	// Refresh butuh GetByID admin lagi.
	mock.ExpectQuery(regexp.QuoteMeta("WHERE a.admin_id = ? LIMIT 1")).
		WithArgs("ADM001").
		WillReturnRows(sqlmock.NewRows(adminSelectColumns).AddRow(
			"ADM001", 1, 1, nil, nil,
			"admin@rms.local", "Super Admin", hash, "active", "yes", "no",
			nil, nil, now, now,
			"Super Admin", "super_admin", "Jakarta",
		))

	newPair, err := svc.Refresh(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Error("expected refresh token rotation - new token must differ from the old one")
	}

	// Refresh token LAMA harus sudah tidak valid lagi (dicabut saat rotasi).
	_, err = svc.Refresh(context.Background(), pair.RefreshToken)
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("expected old refresh token to be invalidated after rotation, got %v", err)
	}
}

func TestAuthService_Refresh_RejectsGarbageToken(t *testing.T) {
	svc, _, _, cleanup := newTestAuthService(t)
	defer cleanup()

	_, err := svc.Refresh(context.Background(), "not-a-real-token")
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("expected ErrInvalidRefreshToken for malformed token, got %v", err)
	}
}

func TestAuthService_IsAccessTokenBlacklisted_FailsOpenWhenRedisDown(t *testing.T) {
	svc, _, mr, cleanup := newTestAuthService(t)
	mr.Close() // matikan Redis SEBELUM cek blacklist
	defer cleanup()

	blacklisted := svc.IsAccessTokenBlacklisted(context.Background(), "some-jti")
	if blacklisted {
		t.Error("expected fail-open (not blacklisted) when Redis is unreachable, since JWT signature+expiry already validated separately")
	}
}
