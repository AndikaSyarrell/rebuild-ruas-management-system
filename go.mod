module rms-backend

go 1.22

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-sql-driver/mysql v1.8.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.5.1
	github.com/xuri/excelize/v2 v2.8.1
	golang.org/x/crypto v0.23.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.3 // indirect
	github.com/xuri/efp v0.0.0-20231025114914-d1ff6096ae53 // indirect
	github.com/xuri/nfp v0.0.0-20230919160717-d98342af3f05 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/text v0.15.0 // indirect
)

// Kedua replace di bawah ini opsional: hanya dibutuhkan bila lingkungan build
// tidak bisa mengakses proxy.golang.org / golang.org langsung (mis. sandbox
// dengan egress terbatas ke GitHub saja). Di lingkungan normal dengan akses
// internet penuh, baris ini aman dihapus - go mod tidy akan tetap resolve ke
// versi yang sama karena mirror GitHub ini identik dengan sumber aslinya.
replace golang.org/x/crypto => github.com/golang/crypto v0.23.0

replace filippo.io/edwards25519 => github.com/FiloSottile/edwards25519 v1.1.0

replace golang.org/x/text => github.com/golang/text v0.14.0

replace golang.org/x/net => github.com/golang/net v0.23.0

replace golang.org/x/image => github.com/golang/image v0.15.0

replace gopkg.in/yaml.v3 => github.com/go-yaml/yaml v3.0.1+incompatible
