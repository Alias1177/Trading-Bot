package main

import (
	"Trading-Bot/config"
	"Trading-Bot/internal/bot"
	"Trading-Bot/internal/db"
	"log"
)

func main() {
	cfg := config.LoadConfig()
	database := db.ConnectDB(cfg)
	defer database.Close() // Закрываем соединение с БД при завершении

	log.Println("Successfully connected to database")
	bot.Start(cfg, database)
}
