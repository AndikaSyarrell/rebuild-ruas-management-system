package middleware

// NOTE: butuh github.com/alicebob/miniredis/v2 dan github.com/redis/go-redis/v9
// (dependency eksternal, keduanya test-only untuk miniredis). Sandbox tempat
// file ini ditulis tidak punya akses ke proxy.golang.org untuk mengunduhnya,
// sehingga belum sempat di-compile/dijalankan di lingkungan tersebut.
// Jalankan `go get github.com/alicebob/miniredis/v2@latest` lalu
// `go test ./internal/middleware/...` di mesin Anda untuk memverifikasi.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return rdb, func() {
		rdb.Close()
		mr.Close()
	}
}

func noopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimit_AllowsRequestsUnderLimit(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	mw := RateLimit(rdb, 5, time.Minute)
	handler := mw(noopHandler())

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/po", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}
}

func TestRateLimit_BlocksRequestsOverLimit(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	mw := RateLimit(rdb, 3, time.Minute)
	handler := mw(noopHandler())

	var lastCode int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/po", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		lastCode = rr.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 after exceeding limit, got %d", lastCode)
	}
}

func TestRateLimit_DifferentIPsTrackedSeparately(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	mw := RateLimit(rdb, 2, time.Minute)
	handler := mw(noopHandler())

	// IP A menghabiskan limitnya.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/po", nil)
		req.RemoteAddr = "10.0.0.3:1"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// IP B (berbeda) harus TIDAK terpengaruh oleh limit IP A.
	req := httptest.NewRequest("GET", "/api/po", nil)
	req.RemoteAddr = "10.0.0.4:1"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected different IP to be unaffected by other IP's limit, got %d", rr.Code)
	}
}

func TestRateLimit_SetsRateLimitHeaders(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	mw := RateLimit(rdb, 10, time.Minute)
	handler := mw(noopHandler())

	req := httptest.NewRequest("GET", "/api/po", nil)
	req.RemoteAddr = "10.0.0.5:1"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-RateLimit-Limit") != "10" {
		t.Errorf("expected X-RateLimit-Limit=10, got %s", rr.Header().Get("X-RateLimit-Limit"))
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header to be set")
	}
}

func TestRateLimit_FailsOpenWhenRedisDown(t *testing.T) {
	// Redis mati -> request TETAP diloloskan (fail-open), bukan diblokir
	// (fail-closed) - keputusan trade-off yang sudah didokumentasikan
	// eksplisit di komentar rate_limit.go.
	rdb, cleanup := newTestRedis(t)
	cleanup() // langsung matikan Redis sebelum request masuk

	mw := RateLimit(rdb, 5, time.Minute)
	handler := mw(noopHandler())

	req := httptest.NewRequest("GET", "/api/po", nil)
	req.RemoteAddr = "10.0.0.6:1"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected fail-open (200) when Redis is unreachable, got %d", rr.Code)
	}
}
