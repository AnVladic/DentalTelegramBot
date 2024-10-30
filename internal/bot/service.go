package bot

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
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

func findTimesheetByDate(day, month, year int, timesheet []crm.TimesheetResponse) *crm.TimesheetResponse {
	for _, entry := range timesheet {
		if entry.PlannedStart.Day() == day &&
			entry.PlannedStart.Month() == time.Month(month) &&
			entry.PlannedStart.Year() == year {
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
	timesheet []crm.TimesheetResponse, currentDate time.Time) tgbotapi.InlineKeyboardMarkup {
	textDayFunc := func(day, month, year int) (string, string) {
		btnText := fmt.Sprintf("%v", day)
		now := time.Now()
		if now.Day() <= day || int(now.Month()) < month || now.Year() < year {
			freeDay := findTimesheetByDate(day, month, year, timesheet)
			if freeDay != nil {
				btnText = fmt.Sprintf("🟢 %v", day)
			}
		}
		return btnText, fmt.Sprintf("%v.%v.%v", year, month, day)
	}

	specialButtonCallbackData := SpecialButtonCallbackData{
		CallbackData: CallbackData{Command: "switch_timesheet_month"},
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
	query *tgbotapi.CallbackQuery, start time.Time, text *string,
) {
	end := time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	timesheet, err := h.dentalProClient.Timesheet(start, end)
	if err != nil {
		_, _ = h.Send(tgbotapi.NewMessage(query.Message.Chat.ID, h.userTexts.InternalError), false)
		logrus.Error(err)
		return
	}

	if text == nil {
		edit := tgbotapi.NewEditMessageReplyMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			h.GenerateTimesheetCalendar(timesheet, start))
		_, _ = h.EditReplyMarkup(edit, true)
	} else {
		edit := tgbotapi.NewEditMessageTextAndMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			*text,
			h.GenerateTimesheetCalendar(timesheet, start))
		_, _ = h.Edit(edit, true)
	}
}

func (h *TelegramBotHandler) ChangeToDoctorsMarkup(message *tgbotapi.Message) {
	doctors, err := h.dentalProClient.DoctorsList()
	if err != nil {
		_, _ = h.Send(tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError), false)
		logrus.Print(err)
		return
	}
	keyboard := tgbotapi.InlineKeyboardMarkup{}
	for _, doctor := range doctors {
		data := TelegramBotDoctorCallbackData{
			CallbackData: CallbackData{"select_doctor"},
			DoctorID:     doctor.ID,
		}
		bytesData, _ := json.Marshal(data)
		title := fmt.Sprintf(
			"%s - %s", doctor.FIO, strings.Join(GetMapValues(doctor.Departments), ", "))
		btn := tgbotapi.NewInlineKeyboardButtonData(title, string(bytesData))
		row := []tgbotapi.InlineKeyboardButton{btn}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}
	response := tgbotapi.NewEditMessageTextAndMarkup(message.Chat.ID, message.MessageID,
		"Пожалуйста, выберите врача для записи. Вы можете выбрать из доступных специалистов ниже 👇",
		keyboard)
	_, _ = h.Edit(response, true)
}
