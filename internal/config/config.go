package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort string
	AppEnv  string

	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string

	RedisAddr string
	RedisPass string
	RedisDB   int

	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration
	JWTIssuer        string

	CORSAllowedOrigins []string

	// Throttling
	LoginMaxAttemptsPerIP    int
	LoginAttemptsPerIPWindow time.Duration
	LoginMaxAttemptsPerEmail int
	LoginLockoutDuration     time.Duration

	PwChangeMaxAttempts     int
	PwChangeLockoutDuration time.Duration

	GlobalRateLimit  int
	GlobalRateWindow time.Duration

	POExportCacheDir string
	POExportTemplate string
	POExportTTL      time.Duration

	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	FrontendBaseURL string
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func Load() *Config {
	return &Config{
		AppPort: getEnv("APP_PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),

		DBHost: getEnv("DB_HOST", "127.0.0.1"),
		DBPort: getEnv("DB_PORT", "3306"),
		DBUser: getEnv("DB_USER", "root"),
		DBPass: getEnv("DB_PASSWORD", ""),
		DBName: getEnv("DB_NAME", "rms"),

		RedisAddr: getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPass: getEnv("REDIS_PASSWORD", ""),
		RedisDB:   getEnvInt("REDIS_DB", 0),

		JWTAccessSecret:  getEnv("JWT_ACCESS_SECRET", ""),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
		JWTAccessTTL:     getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		JWTIssuer:        getEnv("JWT_ISSUER", "rms-backend"),

		CORSAllowedOrigins: []string{getEnv("CORS_ORIGIN", "http://localhost:5173")},

		LoginMaxAttemptsPerIP:    getEnvInt("LOGIN_MAX_ATTEMPTS_PER_IP", 20),
		LoginAttemptsPerIPWindow: getEnvDuration("LOGIN_ATTEMPTS_PER_IP_WINDOW", 15*time.Minute),
		LoginMaxAttemptsPerEmail: getEnvInt("LOGIN_MAX_ATTEMPTS_PER_EMAIL", 5),
		LoginLockoutDuration:     getEnvDuration("LOGIN_LOCKOUT_DURATION", 15*time.Minute),

		PwChangeMaxAttempts:     getEnvInt("PWCHANGE_MAX_ATTEMPTS", 5),
		PwChangeLockoutDuration: getEnvDuration("PWCHANGE_LOCKOUT_DURATION", 15*time.Minute),

		GlobalRateLimit:  getEnvInt("GLOBAL_RATE_LIMIT", 300),
		GlobalRateWindow: getEnvDuration("GLOBAL_RATE_WINDOW", time.Minute),

		POExportCacheDir: getEnv("PO_EXPORT_CACHE_DIR", "./storage/cache/exports"),
		POExportTemplate: getEnv("PO_EXPORT_TEMPLATE_PATH", ""),
		POExportTTL:      getEnvDuration("PO_EXPORT_TTL", 20*time.Minute),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "no-reply@rms.local"),
		SMTPFromName: getEnv("SMTP_FROM_NAME", "Ruas Management System"),

		FrontendBaseURL: getEnv("FRONTEND_BASE_URL", "http://localhost:5173"),
	}
}
