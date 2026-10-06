// Command migrate: tool migrasi database MySQL (satu file).
//
// Perintah:
//   go run ./cmd/migrate create <nama>        buat pasangan file .up.sql & .down.sql
//   go run ./cmd/migrate up [-n N]            jalankan semua migrasi pending (atau N pertama)
//   go run ./cmd/migrate down [-n N]          batalkan N migrasi terakhir (default 1)
//   go run ./cmd/migrate rollback             batalkan seluruh migrasi pada batch terakhir
//   go run ./cmd/migrate status               tampilkan status tiap migrasi
//   go run ./cmd/migrate path                 tampilkan folder migrasi yang terdeteksi
//
// Opsi umum: -path <folder>  (atau env MIGRATIONS_PATH). Jika tidak diisi, folder
// dicari otomatis: ./migrations, ./db/migrations, lalu naik ke direktori induk.
//
// Riwayat tersimpan di tabel schema_migrations (dibuat otomatis).
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"rms-backend/internal/config"
)

const historyTable = "schema_migrations"

var fileRe = regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

type migration struct {
	Version  string
	Name     string
	UpFile   string
	DownFile string
}

type applied struct {
	Version   string
	Name      string
	Batch     int
	AppliedAt time.Time
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	pathFlag := fs.String("path", os.Getenv("MIGRATIONS_PATH"), "folder file migrasi")
	steps := fs.Int("n", 0, "jumlah migrasi (up: 0=semua, down: default 1)")
	// izinkan nama diletakkan sebelum flag: create nama -path x
	args := os.Args[2:]
	var positional []string
	for len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		positional = append(positional, args[0])
		args = args[1:]
	}
	_ = fs.Parse(args)
	positional = append(positional, fs.Args()...)

	config.LoadDotEnv(".env")

	dir, err := resolveDir(*pathFlag, cmd == "create")
	if err != nil {
		fatal(err)
	}

	switch cmd {
	case "path":
		fmt.Println(dir)
		return
	case "create":
		if len(positional) == 0 {
			fatal(errors.New("nama migrasi wajib diisi: migrate create <nama>"))
		}
		if err := createFiles(dir, strings.Join(positional, "_")); err != nil {
			fatal(err)
		}
		return
	case "up", "down", "rollback", "status":
	default:
		usage()
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := openDB()
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	// kunci agar tidak ada dua proses migrasi berjalan bersamaan
	conn, err := db.Conn(ctx)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()
	var got int
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK('rms_migrate', 10)`).Scan(&got); err != nil || got != 1 {
		fatal(errors.New("gagal mendapatkan lock migrasi (ada proses lain yang berjalan?)"))
	}
	defer conn.ExecContext(ctx, `SELECT RELEASE_LOCK('rms_migrate')`)

	if err := ensureHistory(ctx, conn); err != nil {
		fatal(err)
	}
	files, err := loadMigrations(dir)
	if err != nil {
		fatal(err)
	}
	history, err := loadApplied(ctx, conn)
	if err != nil {
		fatal(err)
	}

	switch cmd {
	case "up":
		err = runUp(ctx, conn, files, history, *steps)
	case "down":
		n := *steps
		if n <= 0 {
			n = 1
		}
		err = runDown(ctx, conn, files, history, n)
	case "rollback":
		err = runRollback(ctx, conn, files, history)
	case "status":
		printStatus(files, history)
	}
	if err != nil {
		fatal(err)
	}
}

func usage() {
	fmt.Println(`Penggunaan: migrate <perintah> [opsi]
  create <nama>      buat file migrasi baru
  up [-n N]          jalankan migrasi pending
  down [-n N]        batalkan N migrasi terakhir (default 1)
  rollback           batalkan batch terakhir
  status             lihat status migrasi
  path               tampilkan folder migrasi
Opsi: -path <folder> atau env MIGRATIONS_PATH`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}

// ---------------------------------------------------------------- path

func resolveDir(explicit string, allowCreate bool) (string, error) {
	if explicit != "" {
		if st, err := os.Stat(explicit); err == nil && st.IsDir() {
			return filepath.Abs(explicit)
		}
		if allowCreate {
			if err := os.MkdirAll(explicit, 0o755); err != nil {
				return "", err
			}
			return filepath.Abs(explicit)
		}
		return "", fmt.Errorf("folder migrasi %q tidak ditemukan", explicit)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	candidates := []string{"migrations", filepath.Join("db", "migrations")}
	for cur := wd; ; {
		for _, c := range candidates {
			p := filepath.Join(cur, c)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				return p, nil
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	if allowCreate {
		p := filepath.Join(wd, "migrations")
		if err := os.MkdirAll(p, 0o755); err != nil {
			return "", err
		}
		return p, nil
	}
	return "", errors.New("folder migrasi tidak ditemukan, gunakan -path atau env MIGRATIONS_PATH")
}

// ---------------------------------------------------------------- db

func openDB() (*sql.DB, error) {
	cfg := config.Load()
	// multiStatements=true agar satu file boleh berisi beberapa statement SQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local&multiStatements=true",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("gagal konek database: %w", err)
	}
	return db, nil
}

func ensureHistory(ctx context.Context, c *sql.Conn) error {
	_, err := c.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS `+historyTable+` (
			version     VARCHAR(20)  NOT NULL PRIMARY KEY,
			name        VARCHAR(255) NOT NULL,
			batch       INT          NOT NULL,
			applied_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

func loadApplied(ctx context.Context, c *sql.Conn) (map[string]applied, error) {
	rows, err := c.QueryContext(ctx, `SELECT version, name, batch, applied_at FROM `+historyTable)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]applied{}
	for rows.Next() {
		var a applied
		if err := rows.Scan(&a.Version, &a.Name, &a.Batch, &a.AppliedAt); err != nil {
			return nil, err
		}
		out[a.Version] = a
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------- files

func loadMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	byVersion := map[string]*migration{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := fileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		mg := byVersion[m[1]]
		if mg == nil {
			mg = &migration{Version: m[1], Name: m[2]}
			byVersion[m[1]] = mg
		}
		full := filepath.Join(dir, e.Name())
		if m[3] == "up" {
			mg.UpFile = full
		} else {
			mg.DownFile = full
		}
	}
	out := make([]migration, 0, len(byVersion))
	for _, mg := range byVersion {
		out = append(out, *mg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func createFiles(dir, name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	name = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return errors.New("nama migrasi tidak valid")
	}
	version := time.Now().Format("20060102150405")
	up := filepath.Join(dir, fmt.Sprintf("%s_%s.up.sql", version, name))
	down := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", version, name))
	if err := os.WriteFile(up, []byte("-- Migrasi UP: "+name+"\n\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(down, []byte("-- Migrasi DOWN: "+name+"\n\n"), 0o644); err != nil {
		return err
	}
	fmt.Println("dibuat:", up)
	fmt.Println("dibuat:", down)
	return nil
}

func execFile(ctx context.Context, c *sql.Conn, path string) error {
	if path == "" {
		return errors.New("file migrasi tidak ada")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(b)) == "" {
		return nil
	}
	_, err = c.ExecContext(ctx, string(b))
	return err
}

// ---------------------------------------------------------------- commands

func runUp(ctx context.Context, c *sql.Conn, files []migration, history map[string]applied, limit int) error {
	var pending []migration
	for _, m := range files {
		if _, ok := history[m.Version]; !ok {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		fmt.Println("tidak ada migrasi pending")
		return nil
	}
	if limit > 0 && limit < len(pending) {
		pending = pending[:limit]
	}

	var maxBatch int
	for _, a := range history {
		if a.Batch > maxBatch {
			maxBatch = a.Batch
		}
	}
	batch := maxBatch + 1

	for _, m := range pending {
		fmt.Printf("UP    %s_%s ... ", m.Version, m.Name)
		if err := execFile(ctx, c, m.UpFile); err != nil {
			fmt.Println("GAGAL")
			return fmt.Errorf("%s_%s: %w", m.Version, m.Name, err)
		}
		if _, err := c.ExecContext(ctx,
			`INSERT INTO `+historyTable+` (version, name, batch) VALUES (?, ?, ?)`, m.Version, m.Name, batch); err != nil {
			fmt.Println("GAGAL")
			return fmt.Errorf("catat riwayat %s: %w", m.Version, err)
		}
		fmt.Println("OK")
	}
	fmt.Printf("selesai: %d migrasi dijalankan (batch %d)\n", len(pending), batch)
	return nil
}

// revert membatalkan daftar migrasi (sudah terurut dari yang terbaru).
func revert(ctx context.Context, c *sql.Conn, files []migration, list []applied) error {
	idx := map[string]migration{}
	for _, m := range files {
		idx[m.Version] = m
	}
	for _, a := range list {
		fmt.Printf("DOWN  %s_%s ... ", a.Version, a.Name)
		m, ok := idx[a.Version]
		if !ok {
			fmt.Println("GAGAL")
			return fmt.Errorf("file migrasi versi %s tidak ditemukan di folder", a.Version)
		}
		if err := execFile(ctx, c, m.DownFile); err != nil {
			fmt.Println("GAGAL")
			return fmt.Errorf("%s_%s: %w", a.Version, a.Name, err)
		}
		if _, err := c.ExecContext(ctx, `DELETE FROM `+historyTable+` WHERE version = ?`, a.Version); err != nil {
			fmt.Println("GAGAL")
			return fmt.Errorf("hapus riwayat %s: %w", a.Version, err)
		}
		fmt.Println("OK")
	}
	return nil
}

func sortedApplied(history map[string]applied) []applied {
	out := make([]applied, 0, len(history))
	for _, a := range history {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version > out[j].Version }) // terbaru dulu
	return out
}

func runDown(ctx context.Context, c *sql.Conn, files []migration, history map[string]applied, n int) error {
	list := sortedApplied(history)
	if len(list) == 0 {
		fmt.Println("tidak ada migrasi yang bisa dibatalkan")
		return nil
	}
	if n < len(list) {
		list = list[:n]
	}
	if err := revert(ctx, c, files, list); err != nil {
		return err
	}
	fmt.Printf("selesai: %d migrasi dibatalkan\n", len(list))
	return nil
}

func runRollback(ctx context.Context, c *sql.Conn, files []migration, history map[string]applied) error {
	all := sortedApplied(history)
	if len(all) == 0 {
		fmt.Println("tidak ada migrasi yang bisa di-rollback")
		return nil
	}
	last := all[0].Batch
	for _, a := range all {
		if a.Batch > last {
			last = a.Batch
		}
	}
	var list []applied
	for _, a := range all {
		if a.Batch == last {
			list = append(list, a)
		}
	}
	if err := revert(ctx, c, files, list); err != nil {
		return err
	}
	fmt.Printf("selesai: rollback batch %d (%d migrasi)\n", last, len(list))
	return nil
}

func printStatus(files []migration, history map[string]applied) {
	fmt.Printf("%-16s %-8s %-6s %-20s %s\n", "VERSION", "STATUS", "BATCH", "APPLIED AT", "NAME")
	seen := map[string]bool{}
	for _, m := range files {
		seen[m.Version] = true
		if a, ok := history[m.Version]; ok {
			fmt.Printf("%-16s %-8s %-6d %-20s %s\n", m.Version, "applied", a.Batch, a.AppliedAt.Format("2006-01-02 15:04:05"), m.Name)
		} else {
			fmt.Printf("%-16s %-8s %-6s %-20s %s\n", m.Version, "pending", "-", "-", m.Name)
		}
	}
	for _, a := range sortedApplied(history) {
		if !seen[a.Version] {
			fmt.Printf("%-16s %-8s %-6d %-20s %s (file hilang)\n", a.Version, "applied", a.Batch, a.AppliedAt.Format("2006-01-02 15:04:05"), a.Name)
		}
	}
}
