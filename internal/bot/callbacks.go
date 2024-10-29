package bot

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"time"
)

type TelegramBotDoctorCallbackData struct {
	CallbackData
	DoctorID int64 `json:"d"`
}

func (h *TelegramBotHandler) ShowCalendarCallback(query *tgbotapi.CallbackQuery) {
	var telegramBotDoctorCallbackData TelegramBotDoctorCallbackData
	err := json.Unmarshal([]byte(query.Data), &telegramBotDoctorCallbackData)
	if err != nil {
		logrus.Error(err)
		return
	}

	now := time.Now()
	h.ChangeTimesheet(query, now)
}

func (h *TelegramBotHandler) SwitchTimesheetMonthCallback(query *tgbotapi.CallbackQuery) {
	var specialButtonCallbackData SpecialButtonCallbackData
	err := json.Unmarshal([]byte(query.Data), &specialButtonCallbackData)
	if err != nil {
		logrus.Error(err)
		return
	}

	var year, month int
	_, err = fmt.Sscanf(specialButtonCallbackData.Month, "%d.%d", &year, &month)
	if err != nil {
		logrus.Error(err)
	}
	switch specialButtonCallbackData.Button {
	case BTN_NEXT:
		nextMonth := time.Date(
			year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
		h.ChangeTimesheet(query, nextMonth)
	case BTN_PREV:
		prevMonth := time.Date(
			year, time.Month(month)-1, 1, 0, 0, 0, 0, time.UTC)
		h.ChangeTimesheet(query, prevMonth)
	default:
		logrus.Error("Unknown button")
		return
	}
}
