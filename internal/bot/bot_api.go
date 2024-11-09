package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"time"
)

type ITGBotAPI interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
}

type INow interface {
	Now() time.Time
}

type RealBot struct {
	*tgbotapi.BotAPI
}

type RealNow struct {
}

func (r RealNow) Now() time.Time {
	return time.Now()
}
