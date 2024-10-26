package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"sync"
)

type Router struct {
	bot          *tgbotapi.BotAPI
	tgBotHandler *TelegramBotHandler
	TgChatStates *map[int64]*TelegramBotState
	ChatStatesMu *sync.Mutex
}

func NewRouter(bot *tgbotapi.BotAPI, tgBotHandler *TelegramBotHandler) *Router {
	return &Router{
		bot:          bot,
		tgBotHandler: tgBotHandler,
		TgChatStates: &map[int64]*TelegramBotState{},
		ChatStatesMu: &sync.Mutex{},
	}
}

func (r *Router) StartListening() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := r.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			r.handleMessage(update.Message)
		}
	}
}

func (r *Router) handleMessage(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		r.tgBotHandler.StartCommandHandler(msg)
	case "cancel":
		r.tgBotHandler.CancelCommandHandler(msg)
	default:
		logrus.Printf("Unknown command: %s", msg.Command())
	}
}
