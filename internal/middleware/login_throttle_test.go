package middleware

// NOTE: butuh github.com/alicebob/miniredis/v2 (dependency eksternal, test-only).
// Lihat catatan di rate_limit_test.go - belum sempat di-compile di sandbox ini
// karena proxy.golang.org diblokir. Jalankan `go test ./internal/middleware/...`
// di mesin Anda sendiri untuk memverifikasi.

import (
	"context"
	"testing"
	"time"
)

func newTestLoginThrottle(t *testing.T, maxPerIP int, ipWindow time.Duration, maxPerEmail int, lockout time.Duration) (*LoginThrottle, func()) {
	t.Helper()
	rdb, cleanup := newTestRedis(t)
	return NewLoginThrottle(rdb, maxPerIP, ipWindow, maxPerEmail, lockout), cleanup
}

func TestLoginThrottle_AllowsWithinLimits(t *testing.T) {
	lt, cleanup := newTestLoginThrottle(t, 20, 15*time.Minute, 5, 15*time.Minute)
	defer cleanup()

	allowed, reason, _ := lt.PreCheck(context.Background(), "1.2.3.4", "admin@rms.local")
	if !allowed {
		t.Errorf("expected first attempt to be allowed, got reason=%s", reason)
	}
}

func TestLoginThrottle_LocksAccountAfterMaxFailedAttempts(t *testing.T) {
	lt, cleanup := newTestLoginThrottle(t, 100, 15*time.Minute, 3, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()
	email := "victim@rms.local"

	for i := 0; i < 3; i++ {
		lt.RegisterFailure(ctx, email)
	}

	allowed, reason, _ := lt.PreCheck(ctx, "9.9.9.9", email)
	if allowed {
		t.Error("expected account to be locked after reaching max failed attempts")
	}
	if reason != "account_locked" {
		t.Errorf("expected reason=account_locked, got %s", reason)
	}
}

func TestLoginThrottle_LockPersistsAcrossDifferentIPs(t *testing.T) {
	// Lockout per-EMAIL harus tetap berlaku meski percobaan berikutnya
	// datang dari IP yang berbeda (melindungi dari credential stuffing
	// lintas-IP, bukan cuma satu sumber).
	lt, cleanup := newTestLoginThrottle(t, 100, 15*time.Minute, 2, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()
	email := "target@rms.local"

	lt.RegisterFailure(ctx, email)
	lt.RegisterFailure(ctx, email)

	allowed, _, _ := lt.PreCheck(ctx, "1.1.1.1", email)
	if allowed {
		t.Error("expected lockout to block attempt from IP 1.1.1.1")
	}
	allowed2, _, _ := lt.PreCheck(ctx, "2.2.2.2", email)
	if allowed2 {
		t.Error("expected lockout to also block attempt from a completely different IP (2.2.2.2)")
	}
}

func TestLoginThrottle_BlocksIPAfterTooManyAttempts(t *testing.T) {
	lt, cleanup := newTestLoginThrottle(t, 3, 15*time.Minute, 100, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()
	ip := "5.5.5.5"

	// PreCheck sendiri yang meng-increment counter per-IP (bukan RegisterFailure).
	for i := 0; i < 3; i++ {
		lt.PreCheck(ctx, ip, "someone"+string(rune('a'+i))+"@rms.local")
	}

	allowed, reason, _ := lt.PreCheck(ctx, ip, "yet-another@rms.local")
	if allowed {
		t.Error("expected IP to be rate-limited after exceeding max attempts per IP")
	}
	if reason != "ip_rate_limited" {
		t.Errorf("expected reason=ip_rate_limited, got %s", reason)
	}
}

func TestLoginThrottle_RegisterSuccessClearsFailures(t *testing.T) {
	lt, cleanup := newTestLoginThrottle(t, 100, 15*time.Minute, 3, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()
	email := "recovering@rms.local"

	lt.RegisterFailure(ctx, email)
	lt.RegisterFailure(ctx, email)
	lt.RegisterSuccess(ctx, email) // login berhasil di percobaan ke-3 - counter harus reset

	// Percobaan gagal 2x lagi setelah sukses TIDAK boleh langsung mengunci
	// (karena counter sudah direset), berbeda dari 2+2=4 attempts berturut2.
	lt.RegisterFailure(ctx, email)
	lt.RegisterFailure(ctx, email)

	allowed, _, _ := lt.PreCheck(ctx, "3.3.3.3", email)
	if !allowed {
		t.Error("expected account to remain unlocked after counter reset by RegisterSuccess")
	}
}

func TestLoginThrottle_DifferentEmailsIndependent(t *testing.T) {
	lt, cleanup := newTestLoginThrottle(t, 100, 15*time.Minute, 2, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()

	lt.RegisterFailure(ctx, "userA@rms.local")
	lt.RegisterFailure(ctx, "userA@rms.local")

	// userB sama sekali belum pernah gagal - harus tetap boleh login.
	allowed, _, _ := lt.PreCheck(ctx, "1.1.1.1", "userB@rms.local")
	if !allowed {
		t.Error("expected unrelated email to remain unaffected by userA's lockout")
	}
}
