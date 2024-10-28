package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
	"time"
)

func (h *TelegramBotHandler) Send(
	msgConfig tgbotapi.MessageConfig, errNotifyUser bool) (tgbotapi.Message, error) {
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
	return msg, err
}

func (h *TelegramBotHandler) Edit(
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
	return keyboard
}

func (h *TelegramBotHandler) ChangeTimesheet(query *tgbotapi.CallbackQuery, start time.Time, end time.Time) {
	timesheet, err := h.dentalProClient.Timesheet(start, end)
	if err != nil {
		_, _ = h.Send(tgbotapi.NewMessage(query.Message.Chat.ID, h.userTexts.InternalError), false)
		logrus.Error(err)
		return
	}
	edit := tgbotapi.NewEditMessageReplyMarkup(
		query.Message.Chat.ID, query.Message.MessageID, h.GenerateTimesheetCalendar(timesheet, start))
	_, _ = h.Edit(edit, true)
}
