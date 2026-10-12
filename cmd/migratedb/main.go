// Runner migrasi database (MySQL/MariaDB) menggunakan golang-migrate.
// File SQL migrasi dibaca dari db/migrations (*.up.sql / *.down.sql).
//
// Sumber koneksi (urutan prioritas):
//  1. DATABASE_URL            contoh: mysql://root:@tcp(127.0.0.1:3306)/rebuild_rms
//  2. DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME  (dari environment atau file .env)
//  3. Nilai bawaan XAMPP: 127.0.0.1, 3306, root, (tanpa password), rebuild_rms
//
// Contoh (jalankan dari root project):
//
//	go run ./cmd/migrate -action=version
//	go run ./cmd/migrate -action=up
//	go run ./cmd/migrate -action=down -steps=1
//	go run ./cmd/migrate -action=force -version=1
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	action := flag.String("action", "up", "up | down | version | force")
	steps := flag.Int("steps", 0, "jumlah step untuk up/down (0 = semua)")
	version := flag.Int("version", -1, "versi tujuan untuk action=force")
	path := flag.String("path", "db/migrations", "folder file migrasi")
	envFile := flag.String("env", ".env", "path file .env (diabaikan jika tidak ada)")
	yes := flag.Bool("yes", false, "izinkan down tanpa -steps (membatalkan semua migrasi)")
	flag.Parse()

	loadDotEnv(*envFile)
	dsn := buildDatabaseURL()

	// Tampilkan target (tanpa password) supaya tidak salah database.
	fmt.Printf("Target database : %s\n", describeTarget(dsn))
	fmt.Printf("Folder migrasi  : %s\n\n", *path)

	m, err := migrate.New("file://"+*path, dsn)
	if err != nil {
		log.Fatalf("gagal init migrate: %v", err)
	}
	defer m.Close()

	switch *action {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}

	case "down":
		if *steps <= 0 && !*yes {
			log.Fatal("down tanpa -steps akan membatalkan SEMUA migrasi. Tambahkan -steps=N, atau -yes jika memang disengaja")
		}
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}

	case "force":
		if *version < 0 {
			log.Fatal("action force membutuhkan -version=N (gunakan 0 untuk reset ke kondisi sebelum migrasi pertama)")
		}
		if *version == 0 {
			// golang-migrate memakai -1 sebagai "tidak ada versi"
			err = m.Force(-1)
		} else {
			err = m.Force(*version)
		}

	case "version":
		v, dirty, verr := m.Version()
		if errors.Is(verr, migrate.ErrNilVersion) {
			fmt.Println("belum ada migrasi yang dijalankan")
			return
		}
		if verr != nil {
			log.Fatal(verr)
		}
		fmt.Printf("versi: %d, dirty: %v\n", v, dirty)
		if dirty {
			fmt.Println("status DIRTY: migrasi terhenti di tengah. Periksa DB, rapikan manual, lalu jalankan -action=force -version=<versi terakhir yang sukses>")
		}
		return

	default:
		log.Fatalf("action tidak dikenal: %s", *action)
	}

	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("tidak ada perubahan, DB sudah di versi terbaru")
			return
		}
		log.Fatalf("migrasi gagal: %v", err)
	}
	fmt.Println("migrasi selesai")
}

// buildDatabaseURL menyusun koneksi dari DATABASE_URL, atau dari DB_* / nilai bawaan XAMPP.
func buildDatabaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return ensureMultiStatements(v)
	}

	host := getenv("DB_HOST", "127.0.0.1")
	port := getenv("DB_PORT", "3306")
	user := getenv("DB_USER", "root")
	pass := os.Getenv("DB_PASSWORD") // kosong = valid (bawaan XAMPP)
	name := getenv("DB_NAME", "rebuild_rms")

	dsn := fmt.Sprintf("mysql://%s:%s@tcp(%s:%s)/%s",
		url.QueryEscape(user), url.QueryEscape(pass), host, port, name)
	return ensureMultiStatements(dsn)
}

// File migrasi berisi banyak statement, jadi dibutuhkan multiStatements=true.
func ensureMultiStatements(dsn string) string {
	if strings.Contains(dsn, "multiStatements=") {
		return dsn
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&multiStatements=true"
	}
	return dsn + "?multiStatements=true"
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv membaca file .env sederhana (KEY=VALUE, baris # diabaikan).
// Variabel yang sudah ada di environment tidak ditimpa. Nama kunci dijadikan
// huruf besar, jadi db_host dan DB_HOST sama-sama terbaca.
func loadDotEnv(file string) {
	f, err := os.Open(file)
	if err != nil {
		return // .env tidak ada: tidak masalah
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(line[:i]))
		val := strings.TrimSpace(line[i+1:])
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

// describeTarget menampilkan user@host/db tanpa password dan query.
func describeTarget(dsn string) string {
	s := strings.TrimPrefix(dsn, "mysql://")
	if i := strings.Index(s, "?"); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndex(s, "@"); i >= 0 {
		user := s[:i]
		if j := strings.Index(user, ":"); j >= 0 {
			user = user[:j]
		}
		return user + "@" + s[i+1:]
	}
	return s
}