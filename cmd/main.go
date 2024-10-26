package main

import (
	"main/internal/bot"
	"main/pkg"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	pkg.InitLogger()

	err := godotenv.Load("configs/.env")
	if err != nil {
		logrus.Panicf("Error loading .env file: %v", err)
	}

	DEBUG := os.Getenv("DEBUG") == "true"
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		logrus.Panic("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	tgBot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		logrus.Panic(err)
	}

	tgBot.Debug = DEBUG
	logrus.Printf("Authorized on account %s", tgBot.Self.UserName)

	userTexts := bot.NewUserTexts()
	telegramBotHandler := bot.NewTelegramBotHandler(tgBot, *userTexts)
	router := bot.NewRouter(tgBot, telegramBotHandler)

	go bot.CleanupUserStates(router.ChatStatesMu, router.TgChatStates)
	router.StartListening()
}
