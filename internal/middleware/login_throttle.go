package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoginThrottle mengimplementasikan 2 lapis proteksi brute-force khusus
// endpoint login, terpisah dari rate limit global:
//
//  1. Per-IP: maksimal N percobaan login (sukses maupun gagal) dalam window
//     waktu tertentu. Mencegah 1 IP mencoba banyak email sekaligus.
//  2. Per-email: setelah N kali gagal berturut-turut, akun tersebut dikunci
//     (lockout) selama durasi tertentu, walau dicoba dari IP berbeda.
//     Ini melindungi 1 akun dari credential stuffing lintas-IP.
//
// Counter percobaan email di-reset begitu login berhasil.
type LoginThrottle struct {
	rdb *redis.Client

	maxAttemptsPerIP    int
	attemptsPerIPWindow time.Duration

	maxAttemptsPerEmail int
	lockoutDuration     time.Duration
}

func NewLoginThrottle(rdb *redis.Client, maxAttemptsPerIP int, attemptsPerIPWindow time.Duration, maxAttemptsPerEmail int, lockoutDuration time.Duration) *LoginThrottle {
	return &LoginThrottle{
		rdb:                 rdb,
		maxAttemptsPerIP:    maxAttemptsPerIP,
		attemptsPerIPWindow: attemptsPerIPWindow,
		maxAttemptsPerEmail: maxAttemptsPerEmail,
		lockoutDuration:     lockoutDuration,
	}
}

func ipAttemptKey(ip string) string    { return fmt.Sprintf("throttle:login:ip:%s", ip) }
func emailFailKey(email string) string { return fmt.Sprintf("throttle:login:fail:%s", email) }
func emailLockKey(email string) string { return fmt.Sprintf("throttle:login:lock:%s", email) }

// PreCheck dipanggil SEBELUM proses verifikasi password. Mengembalikan
// (allowed=false, alasan) jika request harus ditolak lebih dulu tanpa
// menyentuh database sama sekali - baik karena IP sudah terlalu sering
// mencoba, maupun karena akun sedang lockout.
func (t *LoginThrottle) PreCheck(ctx context.Context, ip, email string) (allowed bool, reason string, retryAfter time.Duration) {
	// 1. Cek lockout akun (per email) lebih dulu - paling murah & paling ketat.
	lockTTL, err := t.rdb.TTL(ctx, emailLockKey(email)).Result()
	if err == nil && lockTTL > 0 {
		return false, "account_locked", lockTTL
	}

	// 2. Cek jumlah percobaan per IP dalam window berjalan.
	ipKey := ipAttemptKey(ip)
	count, err := t.rdb.Incr(ctx, ipKey).Result()
	if err == nil {
		if count == 1 {
			t.rdb.Expire(ctx, ipKey, t.attemptsPerIPWindow)
		}
		if int(count) > t.maxAttemptsPerIP {
			ttl, _ := t.rdb.TTL(ctx, ipKey).Result()
			return false, "ip_rate_limited", ttl
		}
	}

	return true, "", 0
}

// RegisterFailure dipanggil setelah login gagal (email tidak ditemukan ATAU
// password salah). Menambah counter gagal per-email; jika sudah mencapai
// batas, akun dikunci selama lockoutDuration.
func (t *LoginThrottle) RegisterFailure(ctx context.Context, email string) {
	failKey := emailFailKey(email)
	count, err := t.rdb.Incr(ctx, failKey).Result()
	if err != nil {
		return
	}
	if count == 1 {
		t.rdb.Expire(ctx, failKey, t.lockoutDuration)
	}
	if int(count) >= t.maxAttemptsPerEmail {
		t.rdb.Set(ctx, emailLockKey(email), "1", t.lockoutDuration)
		t.rdb.Del(ctx, failKey)
	}
}

// RegisterSuccess dipanggil setelah login berhasil untuk membersihkan
// counter gagal & lockout (jika ada) supaya tidak menyandera login
// berikutnya yang sah.
func (t *LoginThrottle) RegisterSuccess(ctx context.Context, email string) {
	t.rdb.Del(ctx, emailFailKey(email))
	t.rdb.Del(ctx, emailLockKey(email))
}
