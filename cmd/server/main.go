package main

import (
	"database/sql"
	"log"

	"github.com/buharamanya/gophkeeper/server/api"
	"github.com/buharamanya/gophkeeper/server/app"
	"github.com/buharamanya/gophkeeper/server/config"
	"github.com/buharamanya/gophkeeper/server/storage/postgres"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	// Подключение к PostgreSQL
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Проверка подключения
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to database successfully")

	// Запуск миграций
	if err := postgres.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Инициализация репозиториев
	userRepo := postgres.NewUserRepository(db)
	dataRepo := postgres.NewDataRepository(db)

	// Инициализация сервисов
	authService := app.NewAuthService(userRepo, cfg.JWTSecret)
	dataService := app.NewDataService(dataRepo)

	// Запуск сервера
	handler := api.NewHandler(authService, dataService)

	log.Printf("Starting GophKeeper server on %s", cfg.ServerAddress)
	if err := handler.Start(cfg.ServerAddress); err != nil {
		log.Fatal("Server error:", err)
	}
}
