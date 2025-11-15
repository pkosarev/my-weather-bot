package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type WeatherResponse struct {
	Weather []struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"weather"`
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
}

type ForecastItem struct {
	Dt   int64 `json:"dt"`
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Weather []struct {
		ID int `json:"id"`
	} `json:"weather"`
}

type ForecastResponse struct {
	List []ForecastItem `json:"list"`
	City struct {
		Timezone int `json:"timezone"`
	} `json:"city"`
}

type DayInfo struct {
	MinTemp       float64
	HasPrecip     bool
	Date          time.Time
	IsInitialized bool
}

func GetWeather(lat, lon float64, apiKey string) (WeatherResponse, error) {
	var weatherData WeatherResponse
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=metric&lang=ru", lat, lon, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return weatherData, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&weatherData); err != nil {
		return weatherData, err
	}
	if len(weatherData.Weather) == 0 {
		return weatherData, fmt.Errorf("данные о погоде не получены")
	}
	return weatherData, nil
}

func GetDecision(data WeatherResponse) (string, string) {
	temp := data.Main.Temp
	weatherCode := data.Weather[0].ID
	description := data.Weather[0].Description
	isGoodTemp := temp >= 5.0 && temp <= 32.0
	isGoodWeather := weatherCode >= 800
	recommendation := fmt.Sprintf("Сейчас %.1f°C и %s.", temp, description)
	if isGoodTemp && isGoodWeather {
		return "✅ **Сегодня:** Похоже, да!", recommendation
	}
	if !isGoodTemp {
		if temp < 5.0 {
			recommendation += " Слишком холодно."
		} else {
			recommendation += " Слишком жарко."
		}
	}
	if !isGoodWeather {
		recommendation += " Возможны осадки."
	}
	return "❌ **Сегодня:** Не стоит. ", recommendation
}

func GetForecast(lat, lon float64, apiKey string) (ForecastResponse, error) {
	var forecastData ForecastResponse
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?lat=%f&lon=%f&appid=%s&units=metric&lang=ru", lat, lon, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return forecastData, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&forecastData); err != nil {
		return forecastData, err
	}
	if len(forecastData.List) == 0 {
		return forecastData, fmt.Errorf("данные о прогнозе не получены")
	}
	return forecastData, nil
}

func Analyze4DayForecast(data ForecastResponse) string {
	location := time.FixedZone("API Timezone", data.City.Timezone)
	now := time.Now().In(location)
	todayKey := now.YearDay()
	tomorrowKey := now.Add(24 * time.Hour).YearDay()
	dailyReport := make(map[int]*DayInfo)
	var orderedDays []int
	for _, item := range data.List {
		itemTime := time.Unix(item.Dt, 0).In(location)
		itemDayKey := itemTime.YearDay()
		if itemDayKey == todayKey {
			continue
		}
		hour := itemTime.Hour()
		if hour < 7 || hour > 22 {
			continue
		}
		if _, ok := dailyReport[itemDayKey]; !ok {
			dailyReport[itemDayKey] = &DayInfo{
				MinTemp:       item.Main.Temp,
				HasPrecip:     item.Weather[0].ID < 800,
				Date:          itemTime,
				IsInitialized: true,
			}
			orderedDays = append(orderedDays, itemDayKey)
		} else {
			report := dailyReport[itemDayKey]
			if item.Main.Temp < report.MinTemp {
				report.MinTemp = item.Main.Temp
			}
			if item.Weather[0].ID < 800 {
				report.HasPrecip = true
			}
		}
	}
	var replyStrings []string
	replyStrings = append(replyStrings, "Вот твой прогноз на 4 дня:\n")
	for _, dayKey := range orderedDays {
		info := dailyReport[dayKey]
		var dayName string
		if info.Date.YearDay() == tomorrowKey {
			dayName = "Завтра"
		} else {
			dayName = info.Date.Format("02.01")
		}
		replyStrings = append(replyStrings, formatDecision(dayName, info))
	}
	if len(orderedDays) == 0 {
		return "Не удалось составить прогноз..."
	}
	return (strings.Join(replyStrings, "\n"))
}

func formatDecision(dayName string, info *DayInfo) string {
	if info == nil || !info.IsInitialized {
		return fmt.Sprintf("⚪️ **%s:** Нет данных", dayName)
	}
	isGoodTemp := info.MinTemp >= 5.0
	isGoodWeather := !info.HasPrecip
	precipText := "без осадков"
	if info.HasPrecip {
		precipText = "возможны осадки"
	}
	details := fmt.Sprintf("(Мин. %.0f°C, %s)", info.MinTemp, precipText)
	if isGoodTemp && isGoodWeather {
		return fmt.Sprintf("✅ **%s:** Похоже, да! %s", dayName, details)
	} else {
		return fmt.Sprintf("❌ **%s:** Не стоит. %s", dayName, details)
	}
}