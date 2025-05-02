package main

import (
	"Trading-Bot/internal/binance"
	"Trading-Bot/internal/binance/metrics"
	"Trading-Bot/internal/gpt"
)

func main() {
	// Load environment configuration
	cfg := config.LoadConfig()
	//
	//// Connect to database
	database := db.ConnectDB(cfg)
	defer database.Close() // Close DB connection on exit
	//
	//log.Println("Successfully connected to database")
	//
	//// Start the bot (which will also initialize payment handling)
	//bot.Start(cfg, database)
	candles, err := binance.GetCandles()
	if err != nil {
		panic(err)
	}

	var closes, volumes []float64
	for _, c := range candles {
		closes = append(closes, c.Close)
		volumes = append(volumes, c.Volume)
	}

	rsi := metrics.CalculateRSI(closes[len(closes)-15:], 14)
	ema := metrics.CalculateEMA(closes[len(closes)-50:], 50)
	avgVol := metrics.AverageVolume(volumes[len(volumes)-50:])
	currentVol := volumes[len(volumes)-1]

	prompt := gpt.FormatPrompt(closes, rsi, ema, currentVol, avgVol)
	//fmt.Println("📤 Prompt to model:\n", prompt)
	gpt.AskGPT(prompt,cfg)
}
