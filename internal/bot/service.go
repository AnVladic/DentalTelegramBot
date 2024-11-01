package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
	"main/internal/database"
	"net/http"
	"strings"
	"time"
)

func (h *TelegramBotHandler) Send(
	msgConfig tgbotapi.MessageConfig, errNotifyUser bool) (*tgbotapi.Message, error) {
	msg, err := h.bot.Send(msgConfig)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"chat_id": msgConfig.ChatID,
			"text":    msgConfig.Text,
			"error":   err,
		}).Error("Failed to send message")
		if errNotifyUser {
			response := tgbotapi.NewMessage(msgConfig.ChatID, h.userTexts.InternalError)
			_, _ = h.Send(response, false)
		}
	}
	return &msg, err
}

func (h *TelegramBotHandler) Edit(
	msgConfig tgbotapi.EditMessageTextConfig, errNotifyUser bool) (tgbotapi.Message, error) {
	msg, err := h.bot.Send(msgConfig)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"chat_id": msgConfig.ChatID,
			"text":    msgConfig.ReplyMarkup,
			"error":   err,
		}).Error("Failed to send message")
		if errNotifyUser {
			response := tgbotapi.NewMessage(msgConfig.ChatID, h.userTexts.InternalError)
			_, _ = h.Send(response, false)
		}
	}
	return msg, err
}

func (h *TelegramBotHandler) EditReplyMarkup(
	msgConfig tgbotapi.EditMessageReplyMarkupConfig, errNotifyUser bool) (tgbotapi.Message, error) {
	msg, err := h.bot.Send(msgConfig)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"chat_id": msgConfig.ChatID,
			"text":    msgConfig.ReplyMarkup,
			"error":   err,
		}).Error("Failed to send message")
		if errNotifyUser {
			response := tgbotapi.NewMessage(msgConfig.ChatID, h.userTexts.InternalError)
			_, _ = h.Send(response, false)
		}
	}
	return msg, err
}

func findScheduleByDate(day, month, year int, schedule []crm.WorkSchedule) *crm.WorkSchedule {
	for _, entry := range schedule {
		if entry.Date.Day() == day &&
			entry.Date.Month() == time.Month(month) &&
			entry.Date.Year() == year {
			return &entry
		}
	}
	return nil
}

func (h *TelegramBotHandler) RequestContactKeyboard() tgbotapi.ReplyKeyboardMarkup {
	phoneButton := tgbotapi.KeyboardButton{
		Text:           "📞 Отправить номер телефона",
		RequestContact: true,
	}

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(phoneButton),
	)
	keyboard.OneTimeKeyboard = true
	return keyboard
}

func (h *TelegramBotHandler) RequestPhoneNumber(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.PhoneNumberRequest)
	msg.ReplyMarkup = h.RequestContactKeyboard()
	_, _ = h.Send(msg, true)
}

func (h *TelegramBotHandler) AddBackButton(
	keyboard tgbotapi.InlineKeyboardMarkup, back string) tgbotapi.InlineKeyboardMarkup {
	data := TelegramBackCallback{
		CallbackData{"back"},
		back,
	}
	marshalData, err := json.Marshal(data)
	if err != nil {
		logrus.Error(err)
		return keyboard
	}
	btn := tgbotapi.NewInlineKeyboardButtonData(h.userTexts.Back, string(marshalData))
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tgbotapi.InlineKeyboardButton{btn})
	return keyboard
}

func (h *TelegramBotHandler) GenerateTimesheetCalendar(
	schedule []crm.WorkSchedule, currentDate time.Time, doctorID int64) tgbotapi.InlineKeyboardMarkup {
	textDayFunc := func(day, month, year int) (string, string) {
		btnText := fmt.Sprintf("%v", day)
		now := time.Now()
		if now.Day() <= day || int(now.Month()) < month || now.Year() < year {
			workSchedule := findScheduleByDate(day, month, year, schedule)
			if workSchedule != nil && workSchedule.IsWork {
				btnText = fmt.Sprintf("🟢 %v", day)
			}
		}
		data := TelegramChoiceDayCallback{
			CallbackData{"day"},
			doctorID,
			fmt.Sprintf("%v.%v.%v", year, month, day),
		}
		dataBytes, _ := json.Marshal(data)
		return btnText, string(dataBytes)
	}

	specialButtonCallbackData := SpecialButtonCallbackData{
		CallbackData: CallbackData{Command: "switch_timesheet_month"},
		DoctorID:     doctorID,
	}

	now := time.Now()
	year := currentDate.Year()
	month := currentDate.Month()
	showPrev := now.Year() < year || now.Month() < month
	keyboard := tgbotapi.InlineKeyboardMarkup{}
	keyboard = addMonthYearRow(year, month, keyboard)
	keyboard = addDaysNamesRow(keyboard)
	keyboard = generateMonth(year, int(month), keyboard, textDayFunc)
	keyboard = addSpecialButtons(year, int(month), keyboard, specialButtonCallbackData, showPrev,
		currentDate.Sub(now) < 365*24*time.Hour)
	keyboard = h.AddBackButton(keyboard, "doctors")
	return keyboard
}

func (h *TelegramBotHandler) ChangeTimesheet(
	query *tgbotapi.CallbackQuery, start time.Time, text *string, doctorID int64,
) {
	schedule, err := h.dentalProClient.DoctorWorkSchedule(start, doctorID)
	if err != nil {
		_, _ = h.Send(tgbotapi.NewMessage(query.Message.Chat.ID, h.userTexts.InternalError), false)
		logrus.Error(err)
		return
	}

	if text == nil {
		edit := tgbotapi.NewEditMessageReplyMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			h.GenerateTimesheetCalendar(schedule, start, doctorID))
		_, _ = h.EditReplyMarkup(edit, true)
	} else {
		edit := tgbotapi.NewEditMessageTextAndMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			*text,
			h.GenerateTimesheetCalendar(schedule, start, doctorID))
		_, _ = h.Edit(edit, true)
	}
}

func (h *TelegramBotHandler) ChangeToDoctorsMarkup(message *tgbotapi.Message) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.service",
		"func":   "ChangeToDoctorsMarkup",
	})

	doctors, err := h.dentalProClient.DoctorsList()
	if err != nil {
		_, _ = h.Send(tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError), false)
		log.Error(err)
		return
	}
	keyboard := tgbotapi.InlineKeyboardMarkup{}
	for _, doctor := range doctors {
		data := TelegramBotDoctorCallbackData{
			CallbackData: CallbackData{"select_doctor"},
			DoctorID:     doctor.ID,
		}
		bytesData, _ := json.Marshal(data)

		doctorRepo := database.DoctorRepository{DB: h.db}
		err := doctorRepo.Upsert(database.Doctor{
			ID:  doctor.ID,
			FIO: doctor.FIO,
		})
		if h.checkAndLogError(err, log, message, "") {
			return
		}

		title := fmt.Sprintf(
			"%s - %s", doctor.FIO, strings.Join(GetMapValues(doctor.Departments), ", "))
		btn := tgbotapi.NewInlineKeyboardButtonData(title, string(bytesData))
		row := []tgbotapi.InlineKeyboardButton{btn}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}
	response := tgbotapi.NewEditMessageTextAndMarkup(message.Chat.ID, message.MessageID,
		h.userTexts.ChooseDoctor,
		keyboard)
	_, _ = h.Edit(response, true)
}

// GetOrCreatePatient return: Patient, created, error
func (h *TelegramBotHandler) GetOrCreatePatient(name, surname, phone string) (crm.Patient, bool, error) {
	patient, err := h.dentalProClient.PatientByPhone(phone)
	var reqErr *crm.RequestError
	if errors.As(err, &reqErr) {
		if reqErr.Code != http.StatusNotFound {
			patient, err = h.dentalProClient.CreatePatient(name, surname, phone)
			if err != nil {
				logrus.Error(err)
				return crm.Patient{}, false, err
			}
			return patient, true, nil
		}
	}
	return patient, false, nil
}

func (h *TelegramBotHandler) checkAndLogError(
	err error, log *logrus.Entry, message *tgbotapi.Message, msg string, args ...interface{}) bool {
	if err != nil {
		log.WithError(err).Errorf(msg, args...)
		_, _ = h.Send(tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError), false)
		return true
	}
	return false
}
