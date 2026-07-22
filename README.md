# RMS Backend (Go)

Backend REST API hasil konversi dari sistem PHP (RMS - Ruas Management System),
ditulis ulang di Go dengan skema database ternormalisasi (`rms_normalized.sql`)
dan alur autentikasi JWT + Redis yang sudah ada (`jwt.go`, `password.go`,
`auth.go`, `rate_limit.go`, `login_throttle.go`, `password_change_throttle.go`,
`security_headers.go`, `auth_handler.go`).

Frontend (React + Vite) berjalan terpisah dan mengakses backend ini murni via
REST API (JSON), tidak ada server-side rendering / session cookie PHP lagi.

## Stack

- Go 1.22, `net/http` + [chi](https://github.com/go-chi/chi) router
- MySQL (schema: `rms_normalized.sql`) via `database/sql` + `go-sql-driver/mysql`
- Redis (refresh-token store, access-token blacklist, rate limit, login/password throttling)
- JWT (access + refresh, HS256)

## Menjalankan

```bash
cp .env.example .env
# edit .env: DB_*, REDIS_*, dan WAJIB ganti JWT_ACCESS_SECRET / JWT_REFRESH_SECRET

go mod tidy
go run ./cmd/api
```

Server berjalan di `http://localhost:8080` (lihat `APP_PORT`). Health check:
`GET /healthz`.

Import skema database dari `rms_normalized.sql` ke MySQL sebelum menjalankan.

## Struktur proyek

```
cmd/api/main.go            entrypoint: wiring config, DB, Redis, handler, server
internal/config            baca konfigurasi dari environment variables
internal/db                connection pool MySQL
internal/utils             JWT, bcrypt, response helper, pagination (dipertahankan dari kode awal)
internal/middleware        Auth (JWT), RBAC (RequireAccess), rate limit, login/password throttle, security headers
internal/models            struct domain, mengikuti rms_normalized.sql
internal/repository        akses data (SQL ter-parameterisasi, tanpa string concatenation)
internal/service           AuthService (login/refresh/logout/ganti password), Logger, MailService
internal/handlers          HTTP handler per modul (region, division, unit, ppn, role, access, admin, client, po, activity, document, dashboard, report)
internal/routes            pemetaan endpoint + middleware chain
```

## Perubahan keamanan utama dari versi PHP

1. **SQL Injection tertutup total.** Semua query PHP asli menyisipkan input
   `$_REQUEST`/`$_POST` langsung ke string SQL (`"... WHERE id = '$id'"`).
   Di Go, seluruh query memakai placeholder (`?`) lewat `database/sql`, tidak
   ada satupun string concatenation ke SQL.
2. **Password di-hash dengan bcrypt** (cost 12), bukan disimpan/dibandingkan
   sebagai teks/`encrypt_key()` seperti di PHP.
3. **RBAC ditegakkan di middleware (`RequireAccess`)**, bukan dicek manual
   per-controller dengan pola `if ($acsp_edit == 0) { header("Location:..."); }`
   yang mudah lupa dipasang di salah satu action.
4. **Refresh token & blacklist access-token disimpan di Redis**, sehingga
   logout dan ganti password benar-benar mencabut sesi (PHP asli tidak
   pernah punya mekanisme revoke token/session invalidation yang solid).
5. **Rate limiting global + throttling login/ganti-password** per-IP dan
   per-akun (lihat `middleware/rate_limit.go`, `login_throttle.go`,
   `password_change_throttle.go`) untuk menahan brute force.
6. **Transaksi database** dipakai untuk operasi multi-tabel (buat PO + item
   sekaligus, hitung ulang subtotal/PPN/total) sehingga tidak ada state PO
   "setengah jadi" seperti berpotensi terjadi di PHP (insert item satu per
   satu tanpa transaksi).
7. **FOREIGN KEY + ON DELETE CASCADE** pada skema ternormalisasi menangani
   pembersihan data terkait (item PO, dokumen, aktivitas) otomatis saat PO
   dihapus, menggantikan fungsi manual `delete_atpo_bypo()` dkk di PHP.
8. **Validasi kekuatan password**, validasi tanggal (fail-closed jika format
   tidak dikenali, bukan meloloskan string apa adanya ke query seperti PHP),
   dan pembatasan ukuran/tipe file upload.
9. **Token aktivasi/reset password** dibuat dengan `crypto/rand` (bukan
   `sha1(mt_rand())` yang tidak aman secara kriptografis).
10. **CORS eksplisit** dengan whitelist origin frontend, security headers
    (`X-Content-Type-Options`, `X-Frame-Options`, dst).

## Autentikasi

- `POST /api/auth/login` → `{access_token, refresh_token, expires_in, admin}`
- `POST /api/auth/refresh` → rotasi refresh token
- `POST /api/auth/logout` (butuh `Authorization: Bearer`) → revoke sesi
- `POST /api/auth/reset-password` (butuh login) → ganti password, revoke semua sesi
- `POST /api/auth/forgot-password` / `POST /api/auth/reset-password-code` → alur lupa password
- `POST /api/auth/activate` → aktivasi akun admin baru dari email undangan

Semua endpoint `/api/*` lainnya butuh header `Authorization: Bearer <access_token>`.
Endpoint yang mengubah data juga dibatasi oleh `RequireAccess(slug)` yang mencocokkan
`access_slug` pada tabel `T_Access` sesuai role admin yang login (lihat `internal/routes/routes.go`).

## Testing dengan Postman

Folder `postman/` berisi collection + environment siap import:

- `postman/RMS_Backend_API.postman_collection.json`
- `postman/RMS_Local.postman_environment.json`

### 1. Seed data awal (sekali saja)

Setelah `rms_normalized.sql` diimport, jalankan juga `seed.sql`:

```bash
mysql -u root -p rms < rms_normalized.sql
mysql -u root -p rms < seed.sql
```

`seed.sql` membuat:
- 1 region ("Head Office"), 1 division, 2 unit, 1 PPN (11%)
- role **Super Admin** dengan SEMUA `access_slug` yang dipakai `RequireAccess` di `internal/routes/routes.go`
- 1 admin aktif untuk login pertama kali:
  - email: `admin@rms.local`
  - password: `Admin123!`

Untuk membuat admin lain dengan password custom tanpa lewat alur email aktivasi,
generate hash bcrypt-nya dulu:

```bash
go run ./cmd/hashpw "PasswordBaru123!"
```

lalu masukkan hasilnya ke kolom `admin_password` lewat SQL, atau pakai endpoint
`POST /api/admins` (butuh akses `create_new_admin`) yang alurnya via email aktivasi.

### 2. Import ke Postman

1. Postman → **Import** → pilih kedua file di `postman/`.
2. Di kanan atas Postman, pilih environment **RMS Local**.
3. Pastikan `base_url` di environment sesuai (`http://localhost:8080` secara default).

### 3. Urutan testing

1. Buka folder **Auth** → jalankan **Login** (body sudah terisi
   `admin@rms.local` / `Admin123!`). Response sukses otomatis menyimpan
   `access_token`, `refresh_token`, dan `admin_id` ke environment lewat script
   di tab **Tests** - tidak perlu copy-paste manual.
2. Semua request lain di collection ini sudah diset **Authorization: Bearer
   {{access_token}}** di level Collection, jadi begitu Login berhasil, seluruh
   folder lain (Regions, Clients, Purchase Orders, dst) langsung bisa dipakai.
3. Endpoint yang butuh ID dari response sebelumnya (region_id, client_id,
   po_id, dst) juga sudah punya script kecil di tab **Tests** pada request
   "Create ..." yang otomatis mengisi environment variable terkait - jalankan
   request "Create" dulu sebelum "Update"/"Delete"/"Detail" pada resource yang
   sama.
4. Kalau access token kedaluwarsa (default 15 menit, lihat `JWT_ACCESS_TTL`),
   jalankan request **Refresh Token** di folder Auth - tidak perlu login ulang
   selama refresh token (`JWT_REFRESH_TTL`, default 7 hari) masih berlaku.

### 4. Hal-hal yang perlu diperhatikan

- **403 Forbidden**: admin yang login tidak punya `access_slug` yang
  dibutuhkan endpoint tersebut. Admin dari `seed.sql` (role Super Admin)
  sudah punya semua akses; kalau pakai admin lain, cek isi role-nya lewat
  `GET /api/roles/{id}`.
- **401 Unauthorized**: token belum ada / sudah kedaluwarsa / sudah di-logout.
  Login ulang atau jalankan Refresh Token.
- Body **Create PO** butuh `client_id` (klien yang sudah ada, jalankan
  "Create Client" dulu) atau isi `client_name`/`client_email`/`client_phone`/
  `client_address` dan kosongkan/hapus `client_id` untuk membuat klien baru
  sekaligus.
- Upload gambar/dokumen (`POST /api/admins/{id}/image`,
  `POST /api/po/{id}/documents`) pakai body **form-data**; di Postman, pilih
  tipe field **File** pada key `image`/`file` lalu pilih file dari komputer -
  Postman tidak bisa mengisi file lewat JSON.

## Export PO ke Excel (breakdown per-item)

`GET /api/po/export` — butuh akses `export_po`. Filter (semua opsional, query string):

| Param        | Arti                                          |
|--------------|------------------------------------------------|
| `start_date` | tanggal PO mulai (`YYYY-MM-DD`)                |
| `end_date`   | tanggal PO sampai (`YYYY-MM-DD`)                |
| `division`   | `division_id`                                   |
| `status`     | `open`/`progress`/`prepared`/`complete`/`cancel`, kosong/`all` = semua |
| `region`     | `region_id`                                     |
| `client`     | `client_id`                                     |

Contoh: `GET /api/po/export?start_date=2026-01-01&end_date=2026-12-31&region=1&status=complete`

Hasilnya file `.xlsx` **breakdown per-item** (1 baris = 1 item PO, PO dengan 3
item akan muncul di 3 baris dengan info header PO yang sama berulang).

### Caching

Response di-cache di disk (`PO_EXPORT_CACHE_DIR`, default
`./storage/cache/exports`) selama `PO_EXPORT_TTL` (default **20 menit**),
berdasarkan hash dari kombinasi seluruh filter di atas. Selama TTL:
- Request dengan filter **identik** → langsung disajikan dari file cache,
  tanpa query ulang ke database sama sekali.
- Request dengan filter **berbeda** (walau cuma satu param) → dianggap cache
  terpisah, query baru dijalankan.
- Response menyertakan header `X-Export-Cache: HIT` atau `MISS` supaya mudah
  diverifikasi saat testing.
- Request bersamaan dengan filter identik saat cache kosong/kedaluwarsa hanya
  memicu **satu** query+generate (di-dedupe secara in-process); yang lain
  menunggu hasil yang sama.

Ini didesain untuk single-instance deployment: cache berupa file di disk
lokal, TTL dicek dari `mtime` file (tidak butuh Redis/registry terpisah). Kalau
nanti backend di-scale ke banyak instance, `PO_EXPORT_CACHE_DIR` perlu
dipindah ke shared storage (NFS/S3) atau caching-nya diganti ke Redis supaya
seluruh instance berbagi cache yang sama.

### Pakai template `.xlsx` sendiri

Set `PO_EXPORT_TEMPLATE_PATH` ke path file `.xlsx` kamu. Aturannya:
- Baris pertama (row 1) di template dibaca sebagai **header**, tiap cell-nya
  dicocokkan (case-insensitive) ke daftar label kolom bawaan di bawah.
- Kolom yang cocok akan dipakai **sesuai urutan & subset di template kamu** -
  boleh hanya sebagian kolom, boleh diurutkan ulang, boleh diberi styling
  sendiri (border/warna/lebar kolom template tetap dipertahankan).
- Kolom yang labelnya tidak dikenali diabaikan (tidak diisi apa-apa).
- Kalau template tidak ditemukan/tidak valid, otomatis fallback ke sheet
  bawaan (generate dari nol, header bold + freeze row 1).

Label kolom bawaan yang bisa dipakai di header template (harus sama persis,
tidak case-sensitive): `No PO`, `Invoice`, `Status`, `Tanggal PO`, `Region`,
`Divisi`, `PIC`, `Dibuat Oleh`, `Klien`, `Email Klien`, `Telepon Klien`,
`Sub Client`, `Produk`, `Deskripsi`, `Satuan`, `Qty`, `Harga Satuan`,
`Total Item`, `Subtotal PO`, `PPN (%)`, `Jumlah PPN`, `Total PO`,
`Status Bayar`, `Catatan`.



- Resize/generate thumbnail gambar admin belum diimplementasikan (file asli
  disimpan apa adanya); sebaiknya dipindah ke worker/CDN terpisah.
- Export PDF untuk PO belum diimplementasikan (hanya Excel/.xlsx, lihat
  bagian "Export PO ke Excel" di atas).
- Modul `Address` & `History` pada file PHP asli (`api_address.php`,
  `api_history.php`) tidak memiliki tabel yang jelas di `rms_normalized.sql`
  sehingga belum dikonversi; tambahkan modelnya jika skema tersebut memang
  dipakai di aplikasi produksi.
