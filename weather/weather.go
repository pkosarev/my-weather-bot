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

func SummarizeCurrentWeather(data WeatherResponse) string {
	temp := data.Main.Temp
	description := data.Weather[0].Description
	return fmt.Sprintf("Погода на сегодня: %.1f°C, %s.", temp, description)
}

func Summarize4DayForecast(data ForecastResponse) string {
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

	var summaryStrings []string
	summaryStrings = append(summaryStrings, "Вот сводка погоды на 4 дня:")

	for _, dayKey := range orderedDays {
		info := dailyReport[dayKey]
		var dayName string
		if info.Date.YearDay() == tomorrowKey {
			dayName = "Завтра"
		} else {
			dayName = info.Date.Format("02.01")
		}

		precipText := "без осадков"
		if info.HasPrecip {
			precipText = "возможны осадки"
		}
		summaryStrings = append(summaryStrings,
			fmt.Sprintf("День: %s, Мин. темп. %.0f°C, Осадки: %s.", dayName, info.MinTemp, precipText),
		)
	}

	if len(orderedDays) == 0 {
		return "Не удалось составить прогноз."
	}
	return strings.Join(summaryStrings, "\n")
}
