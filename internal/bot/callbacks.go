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

type TelegramBackCallback struct {
	CallbackData
	Back string `json:"b"`
}

type TelegramChoiceDayCallback struct {
	CallbackData
	Date string `json:"dt"`
}

type TelegramChoiceAppointmentCallback struct {
	CallbackData
	AppointmentID int64 `json:"a"`
}

func (h *TelegramBotHandler) ShowCalendarCallback(query *tgbotapi.CallbackQuery) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.callbacks",
		"func":   "ShowCalendarCallback",
	})

	telegramChoiceAppointmentCallback, err := h.parseChoiceAppointmentCallbackData(query, log)
	if err != nil {
		return
	}

	user, err := h.getOrCreateUser(query.From.ID, query.Message, log)
	if err != nil {
		return
	}

	err = h.updateAppointmentRegister(
		*user, query.Message, telegramChoiceAppointmentCallback.AppointmentID, log)
	if err != nil {
		return
	}

	register, err := h.getRegister(*user, query.Message, log)
	if err != nil {
		return
	}

	doctor, err := h.getCRMDoctor(register.DoctorID, query.Message, log)
	if err != nil {
		return
	}

	now := time.Now()

	text := fmt.Sprintf(
		"%s - %s\n🟢 Доступные дни", h.userTexts.Calendar, doctor.FIO,
	)
	h.ChangeTimesheet(query, now, &text, doctor.ID)
}

func (h *TelegramBotHandler) SwitchTimesheetMonthCallback(query *tgbotapi.CallbackQuery) {
	var specialButtonCallbackData SpecialButtonCallbackData
	err := json.Unmarshal([]byte(query.Data), &specialButtonCallbackData)
	if err != nil {
		logrus.Error(err)
		_ = fmt.Errorf("SwitchTimesheetMonthCallback %w", err)
		return
	}

	var year, month int
	_, err = fmt.Sscanf(specialButtonCallbackData.Month, "%d.%d", &year, &month)
	if err != nil {
		logrus.Error(err)
	}
	newDate := time.Date(
		year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	h.ChangeTimesheet(query, newDate, nil, specialButtonCallbackData.DoctorID)
}

func (h *TelegramBotHandler) ShowAppointments(query *tgbotapi.CallbackQuery) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.callbacks",
		"func":   "ShowAppointments",
	})

	callbackData, err := h.parseDoctorCallbackData(query)
	if h.checkAndLogError(err, log, query.Message, "Unmarshal error") {
		return
	}

	user, err := h.getOrCreateUser(query.From.ID, query.Message, log)
	if err != nil {
		return
	}

	ok := h.upsertRegisterDoctorID(
		user.ID, query.Message.Chat.ID, query.Message.MessageID, callbackData.DoctorID, query.Message, log)
	if !ok {
		return
	}

	appointments, err := h.getAvailableAppointments(user, callbackData.DoctorID, query.Data, log, query.Message)
	if err != nil {
		return
	}

	keyboard := h.createAppointmentButtons(appointments, query, log)

	text := h.userTexts.ChooseAppointments
	if len(keyboard.InlineKeyboard) == 0 {
		text = h.noAppointmentsText(callbackData.DoctorID, query, log)
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, text, keyboard)
	_, _ = h.Edit(edit, true)
}

func (h *TelegramBotHandler) ChoiceDayCallback(query *tgbotapi.CallbackQuery) {
	//log := logrus.WithFields(logrus.Fields{
	//	"module": "bot",
	//	"func":   "ChoiceDayCallback",
	//})
	//h.dentalProClient.
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
