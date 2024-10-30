package main

import (
	"database/sql"
	"fmt"
	"main/internal/bot"
	"main/internal/crm"
	"main/pkg"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func LoadEnv() {
	err := godotenv.Load("configs/.env")
	if err != nil {
		panic(fmt.Errorf("error loading .env file: %w", err))
	}
}

func OpenDB() *sql.DB {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return db
}

func InitTelegramBot(debug bool, dentalProClient crm.IDentalProClient, db *sql.DB) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		panic("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	tgBot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		panic(err)
	}

	tgBot.Debug = debug
	logrus.Printf("Authorized on account %s", tgBot.Self.UserName)

	userTexts := bot.NewUserTexts()
	telegramBotHandler := bot.NewTelegramBotHandler(tgBot, *userTexts, dentalProClient, db)
	router := bot.NewRouter(tgBot, telegramBotHandler)

	go bot.CleanupUserStates(router.ChatStatesMu, router.TgChatStates)
	router.StartListening()
}

func main() {
	pkg.InitLogger()
	LoadEnv()
	db := OpenDB()
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			logrus.Fatal(err)
		}
	}(db)

	DEBUG := os.Getenv("DEBUG") == "true"
	TEST := os.Getenv("TEST") != "false"
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		logrus.Panic("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	dentalProClient := crm.NewDentalProClient(
		os.Getenv("DENTAL_PRO_TOKEN"), os.Getenv("DENTAL_PRO_SECRET"), TEST)
	InitTelegramBot(DEBUG, dentalProClient, db)
}
