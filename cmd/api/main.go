package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"rms-backend/internal/config"
	"rms-backend/internal/db"
	"rms-backend/internal/handlers"
	"rms-backend/internal/middleware"
	"rms-backend/internal/repository"
	"rms-backend/internal/routes"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

func main() {
	config.LoadDotEnv(".env")
	cfg := config.Load()

	if cfg.JWTAccessSecret == "" || cfg.JWTRefreshSecret == "" {
		log.Fatal("JWT_ACCESS_SECRET dan JWT_REFRESH_SECRET wajib diisi. " +
			"Pastikan file .env ada di direktori tempat Anda menjalankan `go run ./cmd/api` " +
			"(root project, sejajar dengan go.mod), atau export manual: " +
			"`export JWT_ACCESS_SECRET=... JWT_REFRESH_SECRET=...` sebelum menjalankan aplikasi.")
	}

	sqlDB, err := db.New(db.Config{
		Host: cfg.DBHost, Port: cfg.DBPort, User: cfg.DBUser, Password: cfg.DBPass, Name: cfg.DBName,
	})
	if err != nil {
		log.Fatalf("gagal konek database: %v", err)
	}
	defer sqlDB.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("gagal konek redis: %v", err)
	}
	defer rdb.Close()

	jwtManager := utils.NewJWTManager(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, cfg.JWTIssuer)
	logger := service.NewLogger()
	mailService := service.NewMailService(service.MailConfig{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Password: cfg.SMTPPassword,
		From: cfg.SMTPFrom, FromName: cfg.SMTPFromName,
	})

	// --- Repositories ---
	regionRepo := repository.NewRegionRepo(sqlDB)
	divisionRepo := repository.NewDivisionRepo(sqlDB)
	unitRepo := repository.NewUnitRepo(sqlDB)
	ppnRepo := repository.NewPpnRepo(sqlDB)
	roleRepo := repository.NewRoleRepo(sqlDB)
	accessRepo := repository.NewAccessRepo(sqlDB)
	adminRepo := repository.NewAdminRepo(sqlDB)
	clientRepo := repository.NewClientRepo(sqlDB)
	poRepo := repository.NewPORepo(sqlDB)
	activityRepo := repository.NewActivityRepo(sqlDB)
	documentRepo := repository.NewDocumentRepo(sqlDB)

	// --- Services ---
	authService := service.NewAuthService(adminRepo, jwtManager, rdb)
	exportService := service.NewExportService(poRepo, service.ExportConfig{
		CacheDir:     cfg.POExportCacheDir,
		TemplatePath: cfg.POExportTemplate,
		TTL:          cfg.POExportTTL,
	})
	loginThrottle := middleware.NewLoginThrottle(rdb, cfg.LoginMaxAttemptsPerIP, cfg.LoginAttemptsPerIPWindow,
		cfg.LoginMaxAttemptsPerEmail, cfg.LoginLockoutDuration)
	pwThrottle := middleware.NewPasswordChangeThrottle(rdb, cfg.PwChangeMaxAttempts, cfg.PwChangeLockoutDuration)

	// --- Handlers ---
	deps := &routes.Dependencies{
		Cfg:         cfg,
		DB:          sqlDB,
		JWTManager:  jwtManager,
		AuthService: authService,
		RDB:         rdb,
		AccessRepo:  accessRepo,

		AuthHandler:      handlers.NewAuthHandler(authService, loginThrottle, pwThrottle, mailService, logger, cfg.FrontendBaseURL, cfg),
		RegionHandler:    handlers.NewRegionHandler(regionRepo),
		DivisionHandler:  handlers.NewDivisionHandler(divisionRepo),
		UnitHandler:      handlers.NewUnitHandler(unitRepo),
		PpnHandler:       handlers.NewPpnHandler(ppnRepo),
		RoleHandler:      handlers.NewRoleHandler(roleRepo, accessRepo),
		AccessHandler:    handlers.NewAccessHandler(accessRepo),
		AdminHandler:     handlers.NewAdminHandler(adminRepo, mailService, "./storage/uploads", cfg.FrontendBaseURL),
		ClientHandler:    handlers.NewClientHandler(clientRepo),
		POHandler:        handlers.NewPOHandler(poRepo, activityRepo, clientRepo, ppnRepo, exportService),
		ActivityHandler:  handlers.NewActivityHandler(activityRepo),
		DocumentHandler:  handlers.NewDocumentHandler(documentRepo, poRepo, activityRepo, "./storage/uploads"),
		DashboardHandler: handlers.NewDashboardHandler(poRepo),
		ReportHandler:    handlers.NewReportHandler(poRepo),
	}

	handler := routes.New(deps)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("rms-backend listening on :%s (env=%s)", cfg.AppPort, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
