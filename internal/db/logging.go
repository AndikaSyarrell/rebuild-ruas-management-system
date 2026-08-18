package db

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"
)

// Querier adalah kontrak minimal yang dibutuhkan layer repository - baik
// *sql.DB maupun LoggingDB sama-sama mengimplementasikannya, sehingga repo
// tidak perlu tahu apakah koneksinya dibungkus logger atau tidak.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// LoggingDB membungkus *sql.DB dan mencatat setiap query (durasi + error)
// lewat slog. Query text DIPOTONG dan args TIDAK pernah dicatat, supaya data
// sensitif (password hash, token) tidak pernah masuk ke log.
type LoggingDB struct {
	*sql.DB
	log *slog.Logger
}

func NewLoggingDB(conn *sql.DB, logger *slog.Logger) *LoggingDB {
	return &LoggingDB{DB: conn, log: logger}
}

const slowQueryThreshold = 300 * time.Millisecond

func (l *LoggingDB) logQuery(ctx context.Context, kind, query string, start time.Time, err error) {
	dur := time.Since(start)
	q := strings.Join(strings.Fields(query), " ")
	if len(q) > 300 {
		q = q[:300] + "..."
	}
	attrs := []any{
		slog.String("kind", kind),
		slog.String("query", q),
		slog.Duration("duration", dur),
	}
	switch {
	case err != nil && err != sql.ErrNoRows:
		l.log.ErrorContext(ctx, "query gagal", append(attrs, slog.String("error", err.Error()))...)
	case dur >= slowQueryThreshold:
		l.log.WarnContext(ctx, "query lambat", attrs...)
	default:
		l.log.DebugContext(ctx, "query dijalankan", attrs...)
	}
}

func (l *LoggingDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := l.DB.QueryContext(ctx, query, args...)
	l.logQuery(ctx, "query", query, start, err)
	return rows, err
}

func (l *LoggingDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := l.DB.QueryRowContext(ctx, query, args...)
	// error dari QueryRow baru diketahui saat .Scan() dipanggil pemanggil,
	// jadi di sini hanya durasi yang bisa dicatat secara akurat.
	l.logQuery(ctx, "query_row", query, start, nil)
	return row
}

func (l *LoggingDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	res, err := l.DB.ExecContext(ctx, query, args...)
	l.logQuery(ctx, "exec", query, start, err)
	return res, err
}