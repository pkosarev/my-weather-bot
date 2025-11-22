package bot

import (
	"fmt"
	"log"

	"my-weather-bot/config"
	"my-weather-bot/llm"
	"my-weather-bot/storage"
	"my-weather-bot/weather"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	systemPrompt    = "Ты — дружелюбный бот-помощник для велосипедистов. Твоя задача - на основе сводки погоды дать короткий, неформальный совет."
	userPromptToday = "Вот сводка погоды:\n%s\n\nТвоя задача:\n1. Напиши ОЧЕНЬ короткий, дружелюбный и неформальный ответ на русском языке.\n2. Скажи, стоит ли кататься на велосипеде СЕГОДНЯ.\n3. Если холодно (ниже ~10C) или есть осадки (не \"ясно\" или \"облачно\"), посоветуй не кататься или одеться очень тепло."
	userPrompt4Day  = "Вот сводка погоды:\n%s\n\nТвоя задача:\n1. Напиши ОЧЕНЬ короткий, дружелюбный и неформальный ответ (2-3 предложения) на русском языке.\n2. Если погода хорошая (тепло и без осадков), порекомендуй кататься.\n3. Если холодно (ниже ~10C) или есть осадки, скажи, что это плохая идея или нужно одеться очень тепло.\n4. Не перечисляй дни по отдельности, дай общую оценку \"ближайших дней\"."
)

type Bot struct {
	api       *tgbotapi.BotAPI
	userState map[int64]string
	cfg       *config.Config
	llmClient *llm.Client
}

func New(api *tgbotapi.BotAPI, cfg *config.Config) *Bot {
	apiKey := cfg.GroqApiKey

	return &Bot{
		api:       api,
		userState: make(map[int64]string),
		cfg:       cfg,
		llmClient: llm.NewClient(apiKey),
	}
}

func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		b.handleUpdate(update)
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	if update.Message.IsCommand() {
		b.handleCommand(update)
		return
	}

	if update.Message.Location != nil {
		b.handleLocation(update)
		return
	}
}

func (b *Bot) handleCommand(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID

	switch update.Message.Command() {
	case "start":
		if err := storage.DeleteLocation(chatID); err != nil {
			log.Printf("Ошибка удаления локации при /start: %v", err)
		}
		log.Printf("Локация для %d удалена по команде /start", chatID)

		msgText := "Привет! Я бот для велосипедистов. 🚴‍♂️\n\n" +
			"Я *забыл* вашу старую геолокацию (если она была).\n\n" +
			"/checkride - проверить погоду данный момент.\n" +
			"/forecast - получить прогноз на ближайшие 4 дня.\n\n" +
			"Обе команды используют вашу сохраненную геолокацию. Если ее нет, я попрошу прислать ее один раз.\n\n" +
			"/forgetlocation - удалить вашу сохраненную геолокацию."
		b.api.Send(tgbotapi.NewMessage(chatID, msgText))

	case "checkride":
		if loc, ok := storage.GetLocation(chatID); ok {
			log.Printf("Использую сохраненную локацию для /checkride (ID: %d)", chatID)
			b.sendTodayAnalysis(chatID, loc.Latitude, loc.Longitude)
		} else {
			b.userState[chatID] = "checkride_saveloc"
			b.api.Send(tgbotapi.NewMessage(chatID, "Отправьте геолокацию (📎), и я запомню ее (для /checkride)."))
		}

	case "forecast":
		if loc, ok := storage.GetLocation(chatID); ok {
			log.Printf("Использую сохраненную локацию для /forecast (ID: %d)", chatID)
			b.sendForecastAnalysis(chatID, loc.Latitude, loc.Longitude)
		} else {
			b.userState[chatID] = "forecast_saveloc"
			b.api.Send(tgbotapi.NewMessage(chatID, "Отправьте геолокацию (📎), и я запомню ее (для /forecast)."))
		}

	case "forgetlocation":
		if err := storage.DeleteLocation(chatID); err != nil {
			log.Printf("Ошибка удаления локации: %v", err)
			b.api.Send(tgbotapi.NewMessage(chatID, "Ошибка при удалении локации."))
		} else {
			b.api.Send(tgbotapi.NewMessage(chatID, "Я удалил вашу геолокацию."))
		}

	default:
		b.api.Send(tgbotapi.NewMessage(chatID, "Я не знаю такой команды."))
	}
}

func (b *Bot) handleLocation(update tgbotapi.Update) {
	location := *update.Message.Location
	chatID := update.Message.Chat.ID
	state, _ := b.userState[chatID]

	if err := storage.SaveLocation(chatID, location); err != nil {
		log.Printf("НЕ УДАЛОСЬ СОХРАНИТЬ ЛОКАЦИЮ: %v", err)
		b.api.Send(tgbotapi.NewMessage(chatID, "Не смог сохранить вашу геолокацию, произошла ошибка базы данных."))
		return
	}

	switch state {
	case "checkride_saveloc":
		b.api.Send(tgbotapi.NewMessage(chatID, "Отлично, я запомнил эту геолокацию! Выполняю /checkride..."))
		b.sendTodayAnalysis(chatID, location.Latitude, location.Longitude)

	case "forecast_saveloc":
		b.api.Send(tgbotapi.NewMessage(chatID, "Отлично, я запомнил эту геолокацию! Выполняю /forecast..."))
		b.sendForecastAnalysis(chatID, location.Latitude, location.Longitude)

	default:
		b.api.Send(tgbotapi.NewMessage(chatID, "Я обновил вашу геолокацию. Готовлю прогноз по умолчанию..."))
		b.sendForecastAnalysis(chatID, location.Latitude, location.Longitude)
	}

	delete(b.userState, chatID)
}

func (b *Bot) sendTodayAnalysis(chatID int64, lat, lon float64) {
	weatherData, err := weather.GetWeather(lat, lon, b.cfg.WeatherApiKey)
	if err != nil {
		log.Println(err)
		b.api.Send(tgbotapi.NewMessage(chatID, "Не смог получить *текущую* погоду :("))
		return
	}

	summary := weather.SummarizeCurrentWeather(weatherData)

	reply, err := b.llmClient.GetAnalysis(systemPrompt, fmt.Sprintf(userPromptToday, summary))
	if err != nil {
		log.Printf("ОШИБКА ВЫЗОВА LLM (sendToday): %v", err)
		reply = fmt.Sprintf("LLM-анализ не удался 🤖. Вот сухая сводка:\n%s", summary)
	}

	b.api.Send(tgbotapi.NewMessage(chatID, reply))
}

func (b *Bot) sendForecastAnalysis(chatID int64, lat, lon float64) {
	forecastData, err := weather.GetForecast(lat, lon, b.cfg.WeatherApiKey)
	if err != nil {
		log.Println(err)
		b.api.Send(tgbotapi.NewMessage(chatID, "Не смог получить *прогноз* :("))
		return
	}

	summary := weather.Summarize4DayForecast(forecastData)
	if summary == "Не удалось составить прогноз." {
		b.api.Send(tgbotapi.NewMessage(chatID, summary))
		return
	}

	reply, err := b.llmClient.GetAnalysis(systemPrompt, fmt.Sprintf(userPrompt4Day, summary))
	if err != nil {
		log.Printf("ОШИБКА ВЫЗОВА LLM (sendForecast): %v", err)
		reply = fmt.Sprintf("LLM-анализ не удался 🤖. Вот сухая сводка:\n\n%s", summary)
	}

	b.api.Send(tgbotapi.NewMessage(chatID, reply))
}
