package storage

import (
	"context"
	"encoding/json"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func Connect() error {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatal("DATABASE_URL не установлен в .env")
	}

	var err error
	DB, err = pgxpool.New(context.Background(), databaseUrl)
	if err != nil {
		return err
	}

	if err = DB.Ping(context.Background()); err != nil {
		return err
	}

	log.Println("Успешно подключено к базе данных!")
	return createTable()
}

func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS user_locations (
		chat_id BIGINT PRIMARY KEY,
		location_data JSONB NOT NULL
	);`

	_, err := DB.Exec(context.Background(), query)
	if err != nil {
		log.Printf("Ошибка создания таблицы: %v", err)
	}
	return err
}

func GetLocation(chatID int64) (tgbotapi.Location, bool) {
	var locData []byte
	var location tgbotapi.Location

	query := "SELECT location_data FROM user_locations WHERE chat_id = $1"

	err := DB.QueryRow(context.Background(), query, chatID).Scan(&locData)
	if err != nil {
		return location, false
	}

	if err = json.Unmarshal(locData, &location); err != nil {
		log.Printf("Ошибка Unmarshal JSON из базы: %v", err)
		return location, false
	}

	return location, true
}

func SaveLocation(chatID int64, location tgbotapi.Location) error {
	locData, err := json.Marshal(location)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO user_locations (chat_id, location_data)
	VALUES ($1, $2)
	ON CONFLICT (chat_id) DO UPDATE
	SET location_data = $2;
	`

	_, err = DB.Exec(context.Background(), query, chatID, locData)
	return err
}

func DeleteLocation(chatID int64) error {
	query := "DELETE FROM user_locations WHERE chat_id = $1"
	_, err := DB.Exec(context.Background(), query, chatID)
	return err
}