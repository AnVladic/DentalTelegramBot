package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/AnVladic/DentalTelegramBot/internal/bot"
	"github.com/AnVladic/DentalTelegramBot/internal/crm"
	"github.com/AnVladic/DentalTelegramBot/pkg"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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

func InitTelegramBot(stopCtx context.Context, dentalProClient crm.IDentalProClient, db *sql.DB, debug bool) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		logrus.Panic("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	tgBot, err := tgbotapi.NewBotAPI(botToken)
	rgBotAPI := &bot.TelegramBotAPI{BotAPI: tgBot}
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

	locStr := os.Getenv("LOCATION")
	if locStr == "" {
		locStr = "Europe/Moscow"
	}
	location, err := time.LoadLocation(locStr)
	if err != nil {
		logrus.Panic(err)
	}

	telegramBotHandler := bot.NewTelegramBotHandler(
		rgBotAPI, *userTexts, dentalProClient, db, branchID, location, bot.RealTimeProvider{},
	)
	router := bot.NewRouter(tgBot, telegramBotHandler, false)
	runServer(stopCtx, router)
}

func runServer(stopCtx context.Context, router *bot.Router) {
	go bot.CleanupUserStates(router.ChatStatesMu, router.TgChatStates)
	go router.StartListening()
	fmt.Println("Server is ready")
	<-stopCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := router.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("shutdown: %s", err)
		return
	}
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	DEBUG := os.Getenv("DEBUG") == "true"
	TEST := os.Getenv("TEST") != "false"
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		logrus.Panic("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logrus.Println("Starting metrics server on :8080")
		logrus.Fatal(http.ListenAndServe(":8080", nil))
	}()

	dentalProClient := crm.NewDentalProClient(
		os.Getenv("DENTAL_PRO_TOKEN"), os.Getenv("DENTAL_PRO_SECRET"), TEST, "internal/crm")
	InitTelegramBot(ctx, dentalProClient, db, DEBUG)
}
