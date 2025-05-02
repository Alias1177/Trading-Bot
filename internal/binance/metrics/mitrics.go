package metrics

func CalculateRSI(closes []float64, period int) float64 {
	var gain, loss float64
	for i := 1; i <= period; i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			gain += diff
		} else {
			loss -= diff
		}
	}

	avgGain := gain / float64(period)
	avgLoss := loss / float64(period)

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func CalculateEMA(closes []float64, period int) float64 {
	k := 2.0 / float64(period+1)
	ema := closes[0]
	for i := 1; i < len(closes); i++ {
		ema = closes[i]*k + ema*(1-k)
	}
	return ema
}

func AverageVolume(volumes []float64) float64 {
	var sum float64
	for _, v := range volumes {
		sum += v
	}
	return sum / float64(len(volumes))
}
