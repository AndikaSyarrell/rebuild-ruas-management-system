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

func newTestPwThrottle(t *testing.T, maxAttempts int, lockout time.Duration) (*PasswordChangeThrottle, func()) {
	t.Helper()
	rdb, cleanup := newTestRedis(t)
	return NewPasswordChangeThrottle(rdb, maxAttempts, lockout), cleanup
}

func TestPasswordChangeThrottle_AllowsInitially(t *testing.T) {
	pt, cleanup := newTestPwThrottle(t, 5, 15*time.Minute)
	defer cleanup()

	allowed, _ := pt.PreCheck(context.Background(), "ADM001")
	if !allowed {
		t.Error("expected first attempt to be allowed")
	}
}

func TestPasswordChangeThrottle_LocksAfterMaxFailures(t *testing.T) {
	pt, cleanup := newTestPwThrottle(t, 3, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()
	adminID := "ADM002"

	for i := 0; i < 3; i++ {
		pt.RegisterFailure(ctx, adminID)
	}

	allowed, retryAfter := pt.PreCheck(ctx, adminID)
	if allowed {
		t.Error("expected admin to be locked out after max failed attempts")
	}
	if retryAfter <= 0 {
		t.Error("expected positive retry-after duration when locked out")
	}
}

func TestPasswordChangeThrottle_LockedByAdminID_NotEmail(t *testing.T) {
	// Berbeda dari LoginThrottle (per-email/IP), PasswordChangeThrottle
	// dikunci per admin_id karena aktor sudah pasti terautentikasi (bukan
	// anonymous login attempt) - test ini memastikan admin lain tidak
	// terpengaruh oleh lockout admin_id yang berbeda.
	pt, cleanup := newTestPwThrottle(t, 2, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()
	pt.RegisterFailure(ctx, "ADM003")
	pt.RegisterFailure(ctx, "ADM003")

	allowedOther, _ := pt.PreCheck(ctx, "ADM004")
	if !allowedOther {
		t.Error("expected a different admin_id to remain unaffected by ADM003's lockout")
	}
}

func TestPasswordChangeThrottle_RegisterSuccessClearsLockout(t *testing.T) {
	pt, cleanup := newTestPwThrottle(t, 2, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()
	adminID := "ADM005"

	pt.RegisterFailure(ctx, adminID)
	pt.RegisterSuccess(ctx, adminID) // reset counter sebelum mencapai limit

	pt.RegisterFailure(ctx, adminID) // 1 kegagalan baru setelah reset

	allowed, _ := pt.PreCheck(ctx, adminID)
	if !allowed {
		t.Error("expected admin to remain unlocked after RegisterSuccess reset the counter")
	}
}

func TestPasswordChangeThrottle_RegisterSuccessAfterLockClearsLock(t *testing.T) {
	pt, cleanup := newTestPwThrottle(t, 2, 15*time.Minute)
	defer cleanup()

	ctx := context.Background()
	adminID := "ADM006"

	pt.RegisterFailure(ctx, adminID)
	pt.RegisterFailure(ctx, adminID) // sekarang locked

	pt.RegisterSuccess(ctx, adminID) // eksplisit membersihkan lock juga, bukan cuma fail counter

	allowed, _ := pt.PreCheck(ctx, adminID)
	if !allowed {
		t.Error("expected RegisterSuccess to clear an active lockout, not just the failure counter")
	}
}
