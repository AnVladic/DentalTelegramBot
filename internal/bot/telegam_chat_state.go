package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sync"
	"time"
)

type TelegramBotState struct {
	NextFunc  *func(msg *tgbotapi.Message)
	Timestamp time.Time
}

func CleanupUserStates(mu *sync.Mutex, states *map[int64]*TelegramBotState) {
	for {
		time.Sleep(1 * time.Hour)
		now := time.Now()
		for chatID, state := range *states {
			if now.Sub(state.Timestamp) > 24*time.Hour {
				mu.Lock()
				delete(*states, chatID) // Удаляем состояния старше 24 часов
				mu.Unlock()
			}
		}
	}
}
