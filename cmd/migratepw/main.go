// cmd/migratepw men-decrypt admin_password lama (AES-256-CBC ala PHP encrypt_key/
// decrypt_key) lalu me-re-hash hasil plaintext-nya memakai bcrypt (utils.HashPassword),
// kemudian menuliskan UPDATE T_Admin SET admin_password = ? WHERE admin_id = ?.
//
// Koneksi database memakai config.LoadDotEnv + config.Load + db.New, mengikuti
// pola yang sama seperti cmd/api. Jalankan dari root project (sejajar go.mod)
// supaya file .env terbaca, atau export kredensial DB manual sebelum menjalankan.
//
// PENTING soal turunan key/iv (mengikuti PERSIS logika PHP openssl_decrypt):
//   - PHP: hash('sha256', $secret_key) menghasilkan STRING HEX 64 karakter.
//     openssl_decrypt tidak men-decode hex ini ke biner - ia hanya memotong string
//     tsb menjadi N byte pertama sesuai kebutuhan cipher (AES-256 -> 32 byte).
//   - Sama untuk IV: substr(hash('sha256', $secret_iv), 0, 16) -> 16 karakter
//     pertama dari hex string dipakai APA ADANYA sebagai 16 byte IV.
//
// Pemakaian:
//
//	go run ./cmd/migratepw -secret-key "warehouse key" -secret-iv "warehouse iv" -dry-run=true
//
// Jalankan dulu dengan -dry-run=true untuk melihat preview (admin_id + apakah decrypt
// sukses) TANPA menyentuh database. Setelah yakin benar, jalankan ulang -dry-run=false.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"rms-backend/internal/config" // TODO: ganti "rms-backend/internal" dengan module path project kamu (lihat go.mod)
	"rms-backend/internal/db"
	"rms-backend/internal/utils"
)

func main() {
	secretKey := flag.String("secret-key", "warehouse key", "warehouse key")
	secretIV := flag.String("secret-iv", "warehouse iv", "warehouse iv")
	dryRun := flag.Bool("dry-run", true, "jika true, hanya menampilkan hasil tanpa UPDATE ke DB")
	testValue := flag.String("test-value", "", "opsional: satu nilai admin_password (base64) untuk uji decrypt saja, lalu keluar")
	showPlaintext := flag.Bool("show-plaintext", false, "PERINGATAN: jika true, plaintext hasil decrypt ikut dicetak ke terminal untuk verifikasi manual. Jangan aktifkan di lingkungan yang ter-log/ter-share.")
	flag.Parse()

	if *showPlaintext {
		fmt.Println("!!! -show-plaintext AKTIF: password asli akan tercetak ke terminal di bawah ini. Pastikan output tidak ter-log/ter-screenshot/ter-share. !!!")
	}

	key, iv := deriveKeyIV(*secretKey, *secretIV)

	// Mode uji cepat: cek satu nilai dulu, tidak perlu koneksi DB.
	if *testValue != "" {
		plain, err := decryptLegacy(*testValue, key, iv)
		if err != nil {
			log.Fatalf("decrypt gagal: %v", err)
		}
		fmt.Printf("decrypt sukses, panjang plaintext: %d karakter\n", len(plain))
		if *showPlaintext {
			fmt.Printf("plaintext hasil decrypt: %q\n", plain)
		}
		return
	}

	// ====== Koneksi DB pakai config .env, sama seperti cmd/api ======
	config.LoadDotEnv(".env")
	cfg := config.Load()

	sqlDB, err := db.New(db.Config{
		Host: cfg.DBHost, Port: cfg.DBPort, User: cfg.DBUser, Password: cfg.DBPass, Name: cfg.DBName,
	})
	if err != nil {
		log.Fatalf("gagal konek database: %v", err)
	}
	defer sqlDB.Close()

	rows, err := sqlDB.Query(`SELECT admin_id, admin_password FROM T_Admin WHERE admin_password IS NOT NULL AND admin_password != ''`)
	if err != nil {
		log.Fatalf("gagal query: %v", err)
	}
	defer rows.Close()

	type pending struct {
		id      string
		newHash string
	}
	var toUpdate []pending
	var failed []string

	for rows.Next() {
		var id, oldPass string
		if err := rows.Scan(&id, &oldPass); err != nil {
			log.Fatalf("scan error: %v", err)
		}

		plain, err := decryptLegacy(oldPass, key, iv)
		if err != nil || plain == "" {
			failed = append(failed, id)
			fmt.Printf("[GAGAL DECRYPT] admin_id=%s err=%v\n", id, err)
			continue
		}

		newHash, err := utils.HashPassword(plain)
		if err != nil {
			log.Fatalf("gagal bcrypt untuk %s: %v", id, err)
		}

		if *showPlaintext {
			fmt.Printf("[OK] admin_id=%s plaintext=%q -> bcrypt siap\n", id, plain)
		} else {
			fmt.Printf("[OK] admin_id=%s plaintext_len=%d -> bcrypt siap\n", id, len(plain))
		}
		toUpdate = append(toUpdate, pending{id: id, newHash: newHash})
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows error: %v", err)
	}

	fmt.Printf("\nRingkasan: %d berhasil di-decrypt, %d gagal.\n", len(toUpdate), len(failed))
	if len(failed) > 0 {
		fmt.Println("admin_id yang gagal (perlu ditangani manual / reset password):")
		for _, id := range failed {
			fmt.Println(" -", id)
		}
	}

	if *dryRun {
		fmt.Println("\n[DRY RUN] Tidak ada perubahan yang ditulis ke database. Jalankan ulang dengan -dry-run=false untuk menerapkan.")
		return
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		log.Fatalf("gagal begin tx: %v", err)
	}
	stmt, err := tx.Prepare(`UPDATE T_Admin SET admin_password = ? WHERE admin_id = ?`)
	if err != nil {
		tx.Rollback()
		log.Fatalf("gagal prepare: %v", err)
	}
	defer stmt.Close()

	for _, p := range toUpdate {
		if _, err := stmt.Exec(p.newHash, p.id); err != nil {
			tx.Rollback()
			log.Fatalf("gagal update %s: %v", p.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("gagal commit: %v", err)
	}
	fmt.Printf("\nBerhasil! %d password admin sudah di-re-hash ke bcrypt.\n", len(toUpdate))
}

// deriveKeyIV meniru persis PHP:
//
//	$key = hash('sha256', $secret_key);                    // hex string 64 char
//	$iv  = substr(hash('sha256', $secret_iv), 0, 16);       // 16 char pertama hex string
//
// openssl_decrypt PHP memotong $key ke 32 byte pertama (AES-256) dan memakai $iv
// (16 karakter ASCII dari hex, BUKAN hasil hex-decode) apa adanya sebagai 16 byte IV.
func deriveKeyIV(secretKey, secretIV string) (key []byte, iv []byte) {
	keyHex := sha256Hex(secretKey) // 64 char
	ivHexFull := sha256Hex(secretIV)

	key = []byte(keyHex[:32])   // 32 byte pertama dari string hex (AES-256 key)
	iv = []byte(ivHexFull[:16]) // 16 byte pertama dari string hex (IV = block size AES)
	return key, iv
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// decryptLegacy menyamai PHP:
//
//	openssl_decrypt(base64_decode($string), "AES-256-CBC", $key, 0, $iv)
//
// CATATAN: sebagian data legacy ternyata ter-base64-encode DUA KALI (base64 dari
// base64). Fungsi ini mencoba 1 layer decode dulu; jika hasilnya bukan kelipatan
// block size AES (berarti bukan ciphertext valid), otomatis coba decode sekali lagi.
func decryptLegacy(b64Cipher string, key, iv []byte) (string, error) {
	cipherBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64Cipher))
	if err != nil {
		return "", fmt.Errorf("base64 decode (layer 1): %w", err)
	}

	if len(cipherBytes) == 0 || len(cipherBytes)%aes.BlockSize != 0 {
		// coba layer kedua: isi hasil decode pertama kemungkinan masih berupa
		// string base64 (double-encoded), decode sekali lagi.
		inner, err2 := base64.StdEncoding.DecodeString(strings.TrimSpace(string(cipherBytes)))
		if err2 != nil {
			return "", fmt.Errorf("panjang ciphertext tidak valid setelah 1 layer decode, dan bukan double-base64 juga: %w", err2)
		}
		cipherBytes = inner
	}

	if len(cipherBytes) == 0 || len(cipherBytes)%aes.BlockSize != 0 {
		return "", errors.New("panjang ciphertext tetap tidak valid meski sudah dicoba double-base64-decode")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plain := make([]byte, len(cipherBytes))
	mode.CryptBlocks(plain, cipherBytes)

	unpadded, err := pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return "", fmt.Errorf("unpad: %w", err)
	}

	return strings.TrimRight(string(unpadded), "\x00"), nil
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("data kosong")
	}
	padLen := int(data[length-1])
	if padLen == 0 || padLen > blockSize || padLen > length {
		return nil, errors.New("padding tidak valid")
	}
	for _, b := range data[length-padLen:] {
		if int(b) != padLen {
			return nil, errors.New("padding tidak konsisten")
		}
	}
	return data[:length-padLen], nil
}