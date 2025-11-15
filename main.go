package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"my-weather-bot/storage"
	"my-weather-bot/weather"
)

var userState = make(map[int64]string)

func startHealthCheckServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Запасной порт для локального запуска
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bot is alive!")
	})

	log.Printf("Starting health check server on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start health check server: %v", err)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Файл .env не найден, используем переменные окружения")
	}
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	weatherApiKey := os.Getenv("WEATHER_API_KEY")

	if telegramToken == "" || weatherApiKey == "" {
		log.Fatal("TELEGRAM_TOKEN или WEATHER_API_KEY не установлены.")
	}

	if err := storage.Connect(); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer storage.DB.Close()

	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	go startHealthCheckServer()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		chatID := update.Message.Chat.ID

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				if err := storage.DeleteLocation(chatID); err != nil {
					log.Printf("Ошибка удаления локации при /start: %v", err)
				}
				log.Printf("Локация для %d удалена по команде /start", chatID)

				msg := tgbotapi.NewMessage(chatID,
					"Привет! Я бот для велосипедистов. 🚴‍♂️\n\n"+
						"Я забыл вашу старую геолокацию (если она была).\n\n"+
						"/checkride - проверить погоду *на сейчас* (попросит геолокацию и запомнит ее).\n"+
						"/forecast - получить прогноз *на 4 дня* (использует сохраненную геолокацию).\n"+
						"/forgetlocation - удалить вашу сохраненную геолокацию.")
				bot.Send(msg)

			case "checkride":
				userState[chatID] = "checkride"
				msg := tgbotapi.NewMessage(chatID,
					"Окей, проверяю *сегодняшнюю* погоду. Отправьте геолокацию (📎).")
				bot.Send(msg)

			case "forecast":
				if loc, ok := storage.GetLocation(chatID); ok {
					log.Printf("Использую сохраненную локацию для %d", chatID)
					forecastData, err := weather.GetForecast(loc.Latitude, loc.Longitude, weatherApiKey)
					if err != nil {
						log.Println(err)
						bot.Send(tgbotapi.NewMessage(chatID, "Не смог получить *прогноз* :("))
					} else {
						reply := weather.Analyze4DayForecast(forecastData)
						bot.Send(tgbotapi.NewMessage(chatID, reply))
					}
				} else {
					userState[chatID] = "forecast_saveloc"
					msg := tgbotapi.NewMessage(chatID,
						"Я не знаю вашу геолокацию.\n\n"+
							"Отправьте ее **один раз**, и я запомню ее для будущих запросов /forecast.")
					bot.Send(msg)
				}

			case "forgetlocation":
				if err := storage.DeleteLocation(chatID); err != nil {
					log.Printf("Ошибка удаления локации: %v", err)
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при удалении локации."))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Я забыл вашу геолокацию."))
				}

			default:
				msg := tgbotapi.NewMessage(chatID, "Я не знаю такой команды.")
				bot.Send(msg)
			}
			continue
		}

		if update.Message.Location != nil {
			location := *update.Message.Location

			state, ok := userState[chatID]
			var reply string

			if ok && state == "checkride" {
				if err := storage.SaveLocation(chatID, location); err != nil {
					log.Printf("НЕ УДАЛОСЬ СОХРАНИТЬ (checkride): %v", err)
				}
				log.Printf("Локация для %d обновлена через /checkride", chatID)

				weatherData, err := weather.GetWeather(location.Latitude, location.Longitude, weatherApiKey)
				if err != nil {
					reply = "Не смог получить *текущую* погоду :("
				} else {
					decision, recommendation := weather.GetDecision(weatherData)
					reply = fmt.Sprintf("%s\n%s", decision, recommendation)
				}
				bot.Send(tgbotapi.NewMessage(chatID, reply))

			} else {
				if err := storage.SaveLocation(chatID, location); err != nil {
					log.Printf("НЕ УДАЛОСЬ СОХРАНИТЬ ЛОКАЦИЮ: %v", err)
					bot.Send(tgbotapi.NewMessage(chatID, "Не смог сохранить вашу геолокацию, произошла ошибка."))
					continue
				}

				if ok && state == "forecast_saveloc" {
					bot.Send(tgbotapi.NewMessage(chatID, "Отлично, я запомнил эту геолокацию!"))
				} else if !ok {
					bot.Send(tgbotapi.NewMessage(chatID, "Я обновил вашу геолокацию. Готовлю прогноз..."))
				}

				log.Printf("Выдаем прогноз для %d (по умолчанию)", chatID)
				forecastData, err := weather.GetForecast(location.Latitude, location.Longitude, weatherApiKey)
				if err != nil {
					reply = "Не смог получить *прогноз* :("
				} else {
					reply = weather.Analyze4DayForecast(forecastData)
				}
				bot.Send(tgbotapi.NewMessage(chatID, reply))
			}

			delete(userState, chatID)
		}
	}
}
