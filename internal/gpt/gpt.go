package gpt

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
)

func FormatPrompt(closes []float64, rsi float64, ema50 float64, currentVol float64, avgVol float64) string {
	trend := "вниз"
	if closes[len(closes)-1] > closes[len(closes)-2] {
		trend = "вверх"
	}

	return fmt.Sprintf(`
Вот краткие метрики по BTC/USDT:
- Последнее движение: %.2f → %.2f (%s)
- RSI (14): %.2f
- EMA (50): %.2f
- Объём за последний час: %.2f
- Средний объём за 50 часов: %.2f

На основе этих данных оцени, куда вероятнее всего пойдёт график в ближайшую  минуту.
Ответь строго в формате:
Направление: вверх/вниз
Пояснение: <1-2 предложения>
`, closes[len(closes)-2], closes[len(closes)-1], trend, rsi, ema50, currentVol, avgVol)
}

func AskGPT(prompt string) {
	client := openai.NewClient("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") // вставь свой ключ
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		},
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
