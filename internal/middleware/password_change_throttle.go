package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PasswordChangeThrottle melindungi endpoint reset/change password dari
// brute-force menebak password lama. Berbeda dari LoginThrottle: di sini
// pelaku SUDAH terautentikasi (punya access token valid), jadi throttle
// dikunci per admin_id (bukan per-email/IP) karena identitas aktor sudah pasti.
type PasswordChangeThrottle struct {
	rdb *redis.Client

	maxAttempts     int
	lockoutDuration time.Duration
}

func NewPasswordChangeThrottle(rdb *redis.Client, maxAttempts int, lockoutDuration time.Duration) *PasswordChangeThrottle {
	return &PasswordChangeThrottle{rdb: rdb, maxAttempts: maxAttempts, lockoutDuration: lockoutDuration}
}

func pwFailKey(adminID string) string { return fmt.Sprintf("throttle:pwchange:fail:%s", adminID) }
func pwLockKey(adminID string) string { return fmt.Sprintf("throttle:pwchange:lock:%s", adminID) }

// PreCheck mengembalikan false jika admin sedang dalam masa lockout ganti
// password (terlalu banyak salah memasukkan password lama).
func (t *PasswordChangeThrottle) PreCheck(ctx context.Context, adminID string) (allowed bool, retryAfter time.Duration) {
	ttl, err := t.rdb.TTL(ctx, pwLockKey(adminID)).Result()
	if err == nil && ttl > 0 {
		return false, ttl
	}
	return true, 0
}

func (t *PasswordChangeThrottle) RegisterFailure(ctx context.Context, adminID string) {
	key := pwFailKey(adminID)
	count, err := t.rdb.Incr(ctx, key).Result()
	if err != nil {
		return
	}
	if count == 1 {
		t.rdb.Expire(ctx, key, t.lockoutDuration)
	}
	if int(count) >= t.maxAttempts {
		t.rdb.Set(ctx, pwLockKey(adminID), "1", t.lockoutDuration)
		t.rdb.Del(ctx, key)
	}
}

func (t *PasswordChangeThrottle) RegisterSuccess(ctx context.Context, adminID string) {
	t.rdb.Del(ctx, pwFailKey(adminID))
	t.rdb.Del(ctx, pwLockKey(adminID))
}
