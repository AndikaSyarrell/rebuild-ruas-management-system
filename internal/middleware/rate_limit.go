package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"rms-backend/internal/utils"
)

// RateLimit adalah middleware global (dipasang di semua route) yang membatasi
// jumlah request per-IP dalam window waktu tertentu memakai algoritma fixed
// window counter di Redis (INCR + EXPIRE). Cocok untuk melindungi seluruh API
// dari flooding/basic DoS, terpisah dari throttling login yang lebih ketat.
func RateLimit(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := utils.ClientIP(r)
			key := fmt.Sprintf("ratelimit:global:%s", ip)

			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Jika Redis down, jangan blokir seluruh traffic - fail open,
				// tapi ini area trade-off; untuk keamanan maksimal bisa diubah fail-closed.
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				rdb.Expire(ctx, key, window)
			}

			ttl, _ := rdb.TTL(ctx, key).Result()
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max0(limit-int(count))))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", int(ttl.Seconds())))

			if int(count) > limit {
				utils.Error(w, http.StatusTooManyRequests, "Terlalu banyak permintaan, silakan coba lagi nanti")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
