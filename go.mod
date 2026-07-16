module rms-backend

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-sql-driver/mysql v1.8.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.5.1
	golang.org/x/crypto v0.23.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

// Kedua replace di bawah ini opsional: hanya dibutuhkan bila lingkungan build
// tidak bisa mengakses proxy.golang.org / golang.org langsung (mis. sandbox
// dengan egress terbatas ke GitHub saja). Di lingkungan normal dengan akses
// internet penuh, baris ini aman dihapus - go mod tidy akan tetap resolve ke
// versi yang sama karena mirror GitHub ini identik dengan sumber aslinya.
replace golang.org/x/crypto => github.com/golang/crypto v0.23.0

replace filippo.io/edwards25519 => github.com/FiloSottile/edwards25519 v1.1.0
