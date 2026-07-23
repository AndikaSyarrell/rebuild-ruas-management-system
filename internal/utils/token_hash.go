package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken menghitung SHA-256 dari token acak (aktivasi/reset password)
// sebelum disimpan ke database. Token asli (raw) tetap yang dikirim ke user
// lewat email/link; yang disimpan di kolom admin_token/admin_reset_code
// hanyalah hash-nya - sehingga kalau database bocor, token itu sendiri
// TIDAK langsung bisa dipakai untuk take-over akun.
//
// Sengaja pakai SHA-256 (bukan bcrypt) karena token ini sudah high-entropy
// (20-24 byte dari crypto/rand, bukan password pilihan user) - tidak butuh
// cost factor/salt lambat seperti bcrypt, cukup hash cepat yang tetap
// membuat rainbow-table/reverse lookup tidak praktis.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}