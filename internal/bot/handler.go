package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type TelegramBotHandler struct {
	bot       *tgbotapi.BotAPI
	userTexts UserTexts
}

func NewTelegramBotHandler(bot *tgbotapi.BotAPI, userTexts UserTexts) *TelegramBotHandler {
	return &TelegramBotHandler{bot: bot, userTexts: userTexts}
}

func (h *TelegramBotHandler) StartCommandHandler(message *tgbotapi.Message) {
	logrus.Print("/start command")
	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Welcome)
	_, _ = h.Send(response, true)
}

func (h *TelegramBotHandler) CancelCommandHandler(message *tgbotapi.Message) {
	logrus.Print("/cancel command")
}
