package service

import (
	"log/slog"
	"os"
)

type LogStatus string

const (
	LogStatusInfo    LogStatus = "info"
	LogStatusSuccess LogStatus = "success"
	LogStatusWarning LogStatus = "warning"
	LogStatusError   LogStatus = "error"
)

// LogEntry adalah catatan audit terstruktur untuk aksi-aksi penting
// (login, logout, ganti password, perubahan status PO, dst).
type LogEntry struct {
	UserID    *string
	Module    string
	Action    string
	Status    LogStatus
	Message   string
	IP        string
	UserAgent string
	Metadata  map[string]any
}

// Logger membungkus slog agar log terstruktur (JSON) dan mudah dikirim ke
// sistem observability (ELK/Datadog/dll) tanpa mengubah kode pemanggil.
type Logger struct {
	l *slog.Logger
}

func NewLogger() *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return &Logger{l: slog.New(handler)}
}

func (lg *Logger) Log(e LogEntry) {
	attrs := []any{
		slog.String("module", e.Module),
		slog.String("action", e.Action),
		slog.String("status", string(e.Status)),
		slog.String("ip", e.IP),
		slog.String("user_agent", e.UserAgent),
	}
	if e.UserID != nil {
		attrs = append(attrs, slog.String("user_id", *e.UserID))
	}
	if e.Metadata != nil {
		attrs = append(attrs, slog.Any("metadata", e.Metadata))
	}

	switch e.Status {
	case LogStatusError:
		lg.l.Error(e.Message, attrs...)
	case LogStatusWarning:
		lg.l.Warn(e.Message, attrs...)
	default:
		lg.l.Info(e.Message, attrs...)
	}
}
