package db

import (
	"Trading-Bot/config"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
)

func ConnectDB(cfg *config.Config) *sqlx.DB {
	// Строка подключения для PostgreSQL
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	connect, err := sqlx.Connect("postgres", connectionString)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Проверка соединения
	if err := connect.Ping(); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	return connect
}
