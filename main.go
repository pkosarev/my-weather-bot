package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"my-weather-bot/bot"
	"my-weather-bot/config"
	"my-weather-bot/storage"
)

func startHealthCheckServer(port string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bot is alive!")
	})
	log.Printf("Starting health check server on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start health check server: %v", err)
	}
}

func main() {
	cfg := config.Load()

	if err := storage.Connect(); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer storage.DB.Close()

	client := &http.Client{
		Timeout: 40 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			MaxIdleConns:      -1,
		},
	}

	api, err := tgbotapi.NewBotAPIWithClient(cfg.TelegramToken, tgbotapi.APIEndpoint, client)
	if err != nil {
		log.Panic(err)
	}
	api.Debug = true
	log.Printf("Авторизован как %s", api.Self.UserName)

	go startHealthCheckServer(cfg.Port)

	myBot := bot.New(api, cfg)
	log.Println("Бот запущен!")
	myBot.Run()
}
