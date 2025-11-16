package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"my-weather-bot/bot"
	"my-weather-bot/storage"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Файл .env не найден, используем переменные окружения")
	}
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	weatherApiKey := os.Getenv("WEATHER_API_KEY")
	groqApiKey := os.Getenv("GROQ_API_KEY")

	if telegramToken == "" || weatherApiKey == "" {
		log.Fatal("TELEGRAM_TOKEN или WEATHER_API_KEY не установлены.")
	}

	if err := storage.Connect(); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer storage.DB.Close()

	api, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Panic(err)
	}
	api.Debug = true
	log.Printf("Авторизован как %s", api.Self.UserName)

	myBot := bot.New(api, weatherApiKey, groqApiKey)
	myBot.Run()
}
