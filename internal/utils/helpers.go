package utils

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
	
)

// Pagination adalah representasi page/item-per-page yang sudah divalidasi.
type Pagination struct {
	Page    int
	PerPage int
	Offset  int
}

// ParsePagination membaca query param "page" & "item" dengan default & batas aman,
// menggantikan pola PHP `isset($_REQUEST['page']) ? ... : ""` yang tidak divalidasi.
func ParsePagination(r *http.Request) Pagination {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("item"))
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 200 {
		perPage = 200 // cegah client meminta seluruh tabel sekaligus
	}
	return Pagination{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}

func TotalPage(totalData, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	if totalData%perPage == 0 {
		return totalData / perPage
	}
	return totalData/perPage + 1
}

// ParseDateParam mengonversi berbagai format tanggal umum ke "Y-m-d".
// Mengembalikan string kosong bila input kosong atau tidak valid (fail-closed,
// bukan meloloskan string mentah ke query seperti pada kode PHP asli).
func ParseDateParam(v string) string {
	if v == "" {
		return ""
	}
	layouts := []string{"2006-01-02", "02/01/2006", "01/02/2006", time.RFC3339, "2006-01-02T15:04:05Z07:00"}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

// RandomHex menghasilkan string hex acak sepanjang n byte, dipakai untuk
// token aktivasi / reset password (menggantikan sha1(mt_rand()) di PHP yang
// tidak cryptographically secure).
func RandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateSequentialID membuat ID dengan prefix + timestamp + random suffix,
// contoh: PO250715-A1B2C3D4. Dipakai untuk PK non-auto-increment (po_id, admin_id, dst).
func GenerateSequentialID(prefix string) string {
	ts := time.Now().Format("060102150405")
	return prefix + ts + "-" + strings.ToUpper(RandomHex(4))
}

func AtoiDefault(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func ParseIDParam(w http.ResponseWriter, raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil {
		Error(w, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return id, true
}
