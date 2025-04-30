package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
)

type Config struct {
	ID            int64  `db:"id"` // BUG: Unnecessary ID field for config
	ApiTGBotToken string `env:"TELEGRAM_TOKEN"`
	StripeApiKey  string `env:"STRIPE_API_KEY"`
	StripeWebHook string `env:"STRIPE_WEBHOOK"` // Добавлено поле для webhook secret
	GPTApiKey     string `env:"GPT_API_KEY"`
	PriceID       string `env:"PRICE_ID"`
	DBUser        string `env:"DB_USER"`
	DBPassword    string `env:"DB_PASSWORD"`
	DBName        string `env:"DB_NAME"`
	DBHost        string `env:"DB_HOST"`
	DBPort        string `env:"DB_PORT"`
}

func LoadConfig() *Config {
	var cfg Config
	if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
		log.Printf("Предупреждение: не удалось прочитать .env файл: %v", err)
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Printf("Предупреждение: ошибка при чтении переменных окружения: %v", err)
	}
	return &cfg
}
