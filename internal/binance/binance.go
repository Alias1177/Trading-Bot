package binance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type Candle struct {
	Close  float64
	Volume float64
}

func GetCandles() ([]Candle, error) {
	alphaAPIKey := "TOXQDL6OUVXSAXXG"
	url := fmt.Sprintf(
		"https://www.alphavantage.co/query?function=FX_INTRADAY&from_symbol=AUD&to_symbol=USD&interval=5min&apikey=%s",
		alphaAPIKey,
	)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var rawData [][]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, err
	}

	var candles []Candle
	for _, c := range rawData {
		closeStr := c[4].(string)
		volStr := c[5].(string)

		closeVal, _ := strconv.ParseFloat(closeStr, 64)
		volVal, _ := strconv.ParseFloat(volStr, 64)

		candles = append(candles, Candle{Close: closeVal, Volume: volVal})
	}
	return candles, nil
}
