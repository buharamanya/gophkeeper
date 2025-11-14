package main

import (
	"database/sql"

	"github.com/buharamanya/gophkeeper/server/api"
	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/buharamanya/gophkeeper/server/config"
	"github.com/buharamanya/gophkeeper/server/storage/postgres"
	_ "github.com/lib/pq"
)

func main() {
	// Инициализация логгера
	logger := config.NewLogger()
	defer logger.Sync()
	sugar := logger.Sugar()

	sugar.Info("Starting GophKeeper server...")

	cfg := config.Load()

	// Валидация конфигурации
	if cfg.JWTSecret == "super-secret-jwt-key-change-in-production" {
		sugar.Warn("Using default JWT secret. Change JWT_SECRET in production!")
	}

	if cfg.EnableTLS {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			sugar.Fatal("TLS enabled but TLS_CERT_FILE or TLS_KEY_FILE not set")
		}
		sugar.Info("TLS mode: ENABLED")
	} else {
		sugar.Info("TLS mode: DISABLED")
	}

	// Подключение к PostgreSQL
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		sugar.Fatalw("Failed to connect to database",
			"error", err,
			"database_url", cfg.DatabaseURL,
		)
	}
	defer db.Close()

	// Проверка подключения
	if err := db.Ping(); err != nil {
		sugar.Fatalw("Failed to ping database", "error", err)
	}
	sugar.Info("Connected to database successfully")

	// Запуск миграций
	if err := postgres.RunMigrations(db); err != nil {
		sugar.Fatalw("Failed to run migrations", "error", err)
	}

	// Инициализация репозиториев
	userRepo := postgres.NewUserRepository(db)
	dataRepo := postgres.NewDataRepository(db)

	// Инициализация сервисов с логгером
	authService := app.NewAuthService(userRepo, cfg.JWTSecret, logger)
	dataService := app.NewDataService(dataRepo, logger)

	// Запуск сервера
	handler := api.NewHandler(authService, dataService, cfg, logger)

	sugar.Infow("Starting GophKeeper server",
		"address", cfg.ServerAddress,
		"tls_enabled", cfg.EnableTLS,
	)

	if err := handler.Start(cfg.ServerAddress); err != nil {
		sugar.Fatalw("Server error", "error", err)
	}
}
