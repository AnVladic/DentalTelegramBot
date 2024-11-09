package main

import (
	"database/sql"
	"fmt"
	"main/internal/bot"
	"main/internal/crm"
	"main/pkg"
	"os"
	"strconv"
	"time"

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
		logrus.Panic("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	tgBot, err := tgbotapi.NewBotAPI(botToken)
	rgBotAPI := &bot.RealBot{BotAPI: tgBot}
	if err != nil {
		logrus.Panic(err)
	}

	branchID, err := strconv.ParseInt(os.Getenv("BRANCH_ID"), 10, 64)
	if err != nil {
		branchID = 3
		logrus.Warnf("BRANCH_ID is nil. Set BRANCH_ID = %d", branchID)
	}

	tgBot.Debug = debug
	logrus.Printf("Authorized on account %s", tgBot.Self.UserName)

	userTexts := bot.NewUserTexts()

	location, err := time.LoadLocation(os.Getenv("LOCATION"))
	if err != nil {
		logrus.Panic(err)
	}

	telegramBotHandler := bot.NewTelegramBotHandler(
		rgBotAPI, *userTexts, dentalProClient, db, branchID, location, bot.RealNow{},
	)
	router := bot.NewRouter(tgBot, telegramBotHandler, false)

	go bot.CleanupUserStates(router.ChatStatesMu, router.TgChatStates)
	fmt.Println("Server is ready")

	stopChan := make(chan struct{})
	router.StartListening(stopChan)
}

func main() {
	pkg.InitLogger("app.log")
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
		os.Getenv("DENTAL_PRO_TOKEN"), os.Getenv("DENTAL_PRO_SECRET"), TEST, "internal/crm")
	InitTelegramBot(DEBUG, dentalProClient, db)
}
