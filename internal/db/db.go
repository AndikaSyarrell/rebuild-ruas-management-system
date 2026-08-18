package db

import (
	"database/sql"
	"fmt"
	"time"
	"log/slog"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// New membuka connection pool ke MySQL. Semua query di layer repository WAJIB
// memakai placeholder ("?") - TIDAK ADA string concatenation ke SQL - untuk
// menutup celah SQL injection yang ada di seluruh kode PHP asli.
func New(cfg Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return conn, nil
}

func NewWithLogging(cfg Config, logger *slog.Logger) (*LoggingDB, error) {
	conn, err := New(cfg)
	if err != nil {
		return nil, err
	}
	return NewLoggingDB(conn, logger), nil
}