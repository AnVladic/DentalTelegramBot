package bot

import (
	"context"
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Router struct {
	bot          TelegramBotAPIWrapper
	tgBotHandler *TelegramBotHandler
	TgChatStates *map[int64]*TelegramChatState
	ChatStatesMu *sync.Mutex
	TestWG       *sync.WaitGroup
	updateWG     *sync.WaitGroup
	stopChan     chan struct{}
}

type CallbackData struct {
	Command string `json:"command"`
}

func NewRouter(bot TelegramBotAPIWrapper, tgBotHandler *TelegramBotHandler, test bool) *Router {
	router := &Router{
		bot:          bot,
		tgBotHandler: tgBotHandler,
		TgChatStates: &map[int64]*TelegramChatState{},
		ChatStatesMu: &sync.Mutex{},
		updateWG:     new(sync.WaitGroup),
		stopChan:     make(chan struct{}, 1),
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

func (r *Router) StartListening() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := r.bot.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			r.updateWG.Add(1)
			go func(update tgbotapi.Update) {
				if r.TestWG != nil {
					defer r.TestWG.Done()
				}

				if update.Message != nil {
					r.handleMessage(update.Message)
				}
				if update.CallbackQuery != nil {
					r.callbackMessage(update.CallbackQuery)
				}
				r.updateWG.Done()
			}(update)

		case <-r.stopChan:
			logrus.Println("Stop Listening")
			return
		}
	}
}

func (r *Router) Shutdown(ctx context.Context) error {
	r.stopChan <- struct{}{}
	done := make(chan struct{})
	go func() {
		r.updateWG.Wait()
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (r *Router) callbackMessage(callbackQuery *tgbotapi.CallbackQuery) {
	var data CallbackData
	chatState := r.GetOrCreateChatState(callbackQuery.Message.Chat.ID)
	callbackData := []byte(callbackQuery.Data)
	err := json.Unmarshal(callbackData, &data)
	if err != nil {
		logrus.Error(err)
	}
	switch data.Command {
	case "switch_timesheet_month":
		r.tgBotHandler.SwitchTimesheetMonthCallback(callbackQuery)
	case "select_doctor":
		r.tgBotHandler.ShowAppointments(callbackQuery)
	case "day":
		r.tgBotHandler.ChoiceDayCallback(callbackQuery)
	case "appointment":
		r.tgBotHandler.ShowCalendarCallback(callbackQuery)
	case "interval":
		r.tgBotHandler.RegisterApproveCallback(callbackQuery, chatState)
	case "change_name":
		r.tgBotHandler.ChangeNameCallback(callbackQuery, chatState)
	case "approve":
		r.tgBotHandler.RegisterCallback(callbackQuery)
	case "del_r":
		r.tgBotHandler.ApproveDeleteRecord(callbackQuery, chatState)
	case "back":
		r.tgBotHandler.BackCallback(callbackQuery)
	default:
		logrus.Errorf("unknown command \"%s\"", data.Command)
	}
}

func (r *Router) handleMessage(msg *tgbotapi.Message) {
	chatState := r.GetOrCreateChatState(msg.Chat.ID)
	currentNextFunc := chatState.NextFunc

	switch msg.Command() {
	case "start":
		r.tgBotHandler.StartCommandHandler(msg, chatState)
	case "record":
		r.tgBotHandler.RegisterCommandHandler(msg, chatState)
	case "change_name":
		r.tgBotHandler.ChangeNameHandler(msg, chatState, nil)
	case "myrecords":
		r.tgBotHandler.ShowRecordsListHandler(msg, chatState)
	case "delete_record":
		r.tgBotHandler.DeleteRecordHandler(msg, chatState)
	case "cancel":
		r.tgBotHandler.CancelCommandHandler(msg, chatState)
	default:
		if chatState.NextFunc != nil {
			(*chatState.NextFunc)(msg, chatState)
		} else {
			r.tgBotHandler.UnknownCommandHandler(msg, chatState)
		}
	}
	if currentNextFunc == chatState.NextFunc {
		chatState.UpdateChatState(nil)
	}
}
