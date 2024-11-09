package bot

import (
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Router struct {
	bot          ITGBotAPI
	tgBotHandler *TelegramBotHandler
	TgChatStates *map[int64]*TelegramChatState
	ChatStatesMu *sync.Mutex
	TestWG       *sync.WaitGroup
}

type CallbackData struct {
	Command string `json:"command"`
}

func NewRouter(bot ITGBotAPI, tgBotHandler *TelegramBotHandler, test bool) *Router {
	router := &Router{
		bot:          bot,
		tgBotHandler: tgBotHandler,
		TgChatStates: &map[int64]*TelegramChatState{},
		ChatStatesMu: &sync.Mutex{},
	}
	if test {
		router.TestWG = new(sync.WaitGroup)
	}
	return router
}

func (r *Router) GetOrCreateChatState(chatID int64) *TelegramChatState {
	r.ChatStatesMu.Lock()
	defer r.ChatStatesMu.Unlock()
	chatState := (*r.TgChatStates)[chatID]
	if chatState == nil {
		chatState = &TelegramChatState{Timestamp: time.Now()}
		(*r.TgChatStates)[chatID] = chatState
	}
	return chatState
}

func (r *Router) StartListening(stopChan chan struct{}) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := r.bot.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			go func() {
				if r.TestWG != nil {
					defer r.TestWG.Done()
				}

				if update.Message != nil {
					r.handleMessage(update.Message)
				}
				if update.CallbackQuery != nil {
					var data CallbackData
					chatState := r.GetOrCreateChatState(update.CallbackQuery.Message.Chat.ID)
					callbackData := []byte(update.CallbackQuery.Data)
					err := json.Unmarshal(callbackData, &data)
					if err != nil {
						logrus.Error(err)
					}
					switch data.Command {
					case "switch_timesheet_month":
						r.tgBotHandler.SwitchTimesheetMonthCallback(update.CallbackQuery)
					case "select_doctor":
						r.tgBotHandler.ShowAppointments(update.CallbackQuery)
					case "day":
						r.tgBotHandler.ChoiceDayCallback(update.CallbackQuery)
					case "appointment":
						r.tgBotHandler.ShowCalendarCallback(update.CallbackQuery)
					case "interval":
						r.tgBotHandler.RegisterApproveCallback(update.CallbackQuery, chatState)
					case "change_name":
						r.tgBotHandler.ChangeNameCallback(update.CallbackQuery, chatState)
					case "approve":
						r.tgBotHandler.RegisterCallback(update.CallbackQuery)
					case "back":
						r.tgBotHandler.BackCallback(update.CallbackQuery)
					default:
						logrus.Errorf("unknown command \"%s\"", data.Command)
					}
				}
			}()

		case <-stopChan:
			logrus.Println("Stop Listening")
			return
		}
	}
}

func (r *Router) handleMessage(msg *tgbotapi.Message) {
	chatState := r.GetOrCreateChatState(msg.Chat.ID)

	switch msg.Command() {
	case "start":
		r.tgBotHandler.StartCommandHandler(msg, chatState)
	case "register":
		r.tgBotHandler.RegisterCommandHandler(msg, chatState)
	case "change_name":
		r.tgBotHandler.ChangeNameHandler(msg, chatState, nil)
	case "myrecords":
		r.tgBotHandler.ShowRecordsListHandler(msg, chatState)
	case "cancel":
		r.tgBotHandler.CancelCommandHandler(msg, chatState)
	default:
		if chatState.NextFunc != nil {
			(*chatState.NextFunc)(msg, chatState)
		} else {
			r.tgBotHandler.UnknownCommandHandler(msg, chatState)
		}
	}
}
