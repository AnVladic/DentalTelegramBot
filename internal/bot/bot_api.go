package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"time"
)

type TelegramBotAPIWrapper interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
}

type TimeProvider interface {
	Now() time.Time
}

type TelegramBotAPI struct {
	*tgbotapi.BotAPI
}

type RealTimeProvider struct {
}

func (r RealTimeProvider) Now() time.Time {
	return time.Now()
}
