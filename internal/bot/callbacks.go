package bot

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
	"main/internal/database"
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
	DoctorID int64  `json:"d"`
	Date     string `json:"dt"`
}

type TelegramChoiceAppointmentCallback struct {
	CallbackData
	AppointmentID int64 `json:"a"`
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
		"%s - %s\n🟢 Доступные дни", h.userTexts.Calendar, doctor.FIO,
	)
	h.ChangeTimesheet(query, now, &text, telegramBotDoctorCallbackData.DoctorID)
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
	var telegramBotDoctorCallbackData TelegramBotDoctorCallbackData
	err := json.Unmarshal([]byte(query.Data), &telegramBotDoctorCallbackData)
	if h.checkAndLogError(err, log, query.Message, "Unmarshal error") {
		return
	}

	repository := database.UserRepository{DB: h.db}
	user, _, err := repository.GetOrCreateByTelegramID(database.User{TgUserID: query.From.ID})
	if h.checkAndLogError(err, log, query.Message, "GetOrCreateByTelegramID Unknown error") {
		return
	}
	registerRepo := database.RegisterRepository{DB: h.db}
	_, err = registerRepo.UpsertDoctorID(database.Register{
		UserID:    user.ID,
		ChatID:    query.Message.Chat.ID,
		MessageID: query.Message.MessageID,
		DoctorID:  &telegramBotDoctorCallbackData.DoctorID,
	})
	if h.checkAndLogError(
		err, log,
		query.Message, "UpsertDoctorID %d", telegramBotDoctorCallbackData.DoctorID) {
		return
	}

	var dentalProClientID int64 = 1
	if user.DentalProID != nil && *user.DentalProID > 0 {
		dentalProClientID = *user.DentalProID
	}
	appointments, err := h.dentalProClient.AvailableAppointments(
		dentalProClientID, []int64{telegramBotDoctorCallbackData.DoctorID}, false)
	if h.checkAndLogError(err, log, query.Message, "Get Appointments error, %s", query.Data) {
		return
	}
	keyboard := tgbotapi.InlineKeyboardMarkup{}
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0)
	for _, doctorAppointments := range appointments {
		for _, appointment := range doctorAppointments {
			data, err := json.Marshal(TelegramChoiceAppointmentCallback{
				CallbackData{"appointment"},
				appointment.ID,
			})
			if h.checkAndLogError(err, log, query.Message, "Marshal error") {
				return
			}

			text := fmt.Sprintf("(%d мин.) %s", appointment.Time, appointment.Name)
			button := tgbotapi.NewInlineKeyboardButtonData(text, string(data))
			buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
		}
	}
	keyboard.InlineKeyboard = buttons
	keyboard = h.AddBackButton(keyboard, "doctors")
	if len(buttons) == 0 {
		doctorRepo := database.DoctorRepository{DB: h.db}
		doctor, err := doctorRepo.Get(telegramBotDoctorCallbackData.DoctorID)
		if h.checkAndLogError(err, log, query.Message, "Get Doctor ByID error, %s", query.Data) {
			return
		}
		text := fmt.Sprintf(h.userTexts.DontHasAppointments, doctor.FIO)
		edit := tgbotapi.NewEditMessageTextAndMarkup(
			query.Message.Chat.ID, query.Message.MessageID, text, keyboard,
		)
		_, _ = h.Edit(edit, true)
		return
	}
	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID, query.Message.MessageID, h.userTexts.ChooseAppointments, keyboard,
	)
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
