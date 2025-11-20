package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	WeatherApiKey string
	GroqApiKey    string
	DatabaseUrl   string
	Port          string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		WeatherApiKey: os.Getenv("WEATHER_API_KEY"),
		GroqApiKey:    os.Getenv("GROQ_API_KEY"),
		DatabaseUrl:   os.Getenv("DATABASE_URL"),
		Port:          os.Getenv("PORT"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.TelegramToken == "" || cfg.WeatherApiKey == "" {
		log.Fatal("Ошибка конфига: TELEGRAM_TOKEN или WEATHER_API_KEY не установлены.")
	}

	return cfg
}
