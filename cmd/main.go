package main

import (
	"Trading-Bot/config"
	"Trading-Bot/internal/bot"
	"Trading-Bot/internal/db"
	"log"
)

func main() {
	// Load environment configuration
	cfg := config.LoadConfig()

	// Connect to database
	database := db.ConnectDB(cfg)
	defer database.Close() // Close DB connection on exit

	log.Println("Successfully connected to database")

	// Start the bot (which will also initialize payment handling)
	bot.Start(cfg, database)
}
