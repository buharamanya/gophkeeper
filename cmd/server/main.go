package main

import (
	"database/sql"
	"log"

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

}
