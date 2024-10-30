package bot

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
	"time"
)

type TelegramBotDoctorCallbackData struct {
	CallbackData
	DoctorID int64 `json:"d"`
}

type TelegramBackCallback struct {
	CallbackData
	Back string `json:"b"`
}

func (h *TelegramBotHandler) ShowCalendarCallback(query *tgbotapi.CallbackQuery) {
	var telegramBotDoctorCallbackData TelegramBotDoctorCallbackData
	err := json.Unmarshal([]byte(query.Data), &telegramBotDoctorCallbackData)
	if err != nil {
		logrus.Error(err)
		return
	}

	doctors, err := h.dentalProClient.DoctorsList()
	if err != nil {
		logrus.Error(err)
		response := tgbotapi.NewMessage(query.Message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
	}
	doctor := crm.GetDoctorByID(doctors, telegramBotDoctorCallbackData.DoctorID)
	if doctor == nil {
		logrus.Errorf("Doctor By ID %d Not Found", telegramBotDoctorCallbackData.DoctorID)
		response := tgbotapi.NewMessage(query.Message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return
	}
	now := time.Now()

	text := fmt.Sprintf(
		"%s - %s\n🟢 Рабочие дни", h.userTexts.Calendar, (*doctor).FIO)
	h.ChangeTimesheet(query, now, &text)
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
		h.ChangeTimesheet(query, nextMonth, nil)
	case BTN_PREV:
		prevMonth := time.Date(
			year, time.Month(month)-1, 1, 0, 0, 0, 0, time.UTC)
		h.ChangeTimesheet(query, prevMonth, nil)
	default:
		logrus.Error("Unknown button")
		return
	}
}

func (h *TelegramBotHandler) BackCallback(query *tgbotapi.CallbackQuery) {
	var backCallback TelegramBackCallback
	err := json.Unmarshal([]byte(query.Data), &backCallback)
	if err != nil {
		logrus.Error(err)
		return
	}

	switch backCallback.Back {
	case "doctors":
		h.ChangeToDoctorsMarkup(query.Message)
	}
}
