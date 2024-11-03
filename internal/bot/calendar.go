package bot

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const BTN_PREV = "<"
const BTN_NEXT = ">"

func GenerateCalendar(year int, month time.Month, doctorID int64) tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.InlineKeyboardMarkup{}
	keyboard = addMonthYearRow(year, month, keyboard)
	keyboard = addDaysNamesRow(keyboard)
	keyboard = generateMonth(year, int(month), keyboard, nil)
	keyboard = addSpecialButtons(year, int(month), keyboard, TelegramCalendarSpecialButtonCallback{
		CallbackData: CallbackData{Command: "switch_timesheet_month"},
		DoctorID:     doctorID,
	}, true, true)
	return keyboard
}

func addMonthYearRow(year int, month time.Month, keyboard tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	var row []tgbotapi.InlineKeyboardButton
	btn := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %v", month, year), "1")
	row = append(row, btn)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	return keyboard
}

func addDaysNamesRow(keyboard tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	days := [7]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
	var rowDays []tgbotapi.InlineKeyboardButton
	for _, day := range days {
		btn := tgbotapi.NewInlineKeyboardButtonData(day, day)
		rowDays = append(rowDays, btn)
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rowDays)
	return keyboard
}

func generateMonth(
	year int, month int, keyboard tgbotapi.InlineKeyboardMarkup,
	textDayFunc func(day, month, year int) (string, string),
) tgbotapi.InlineKeyboardMarkup {

	if textDayFunc == nil {
		textDayFunc = func(day, month, year int) (string, string) {
			btnText := fmt.Sprintf("%v", day)
			if time.Now().Day() == day {
				btnText = fmt.Sprintf("%v!", day)
			}
			return btnText, fmt.Sprintf("%v.%v.%v", year, month, day)
		}
	}

	firstDay := _date(year, month, 0)
	amountDaysInMonth := _date(year, month+1, 0).Day()

	weekday := int(firstDay.Weekday())
	var rowDays []tgbotapi.InlineKeyboardButton
	for i := 1; i <= weekday; i++ {
		btn := tgbotapi.NewInlineKeyboardButtonData(" ", string(i))
		rowDays = append(rowDays, btn)
	}

	amountWeek := weekday
	for i := 1; i <= amountDaysInMonth; i++ {
		if amountWeek == 7 {
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rowDays)
			amountWeek = 0
			rowDays = []tgbotapi.InlineKeyboardButton{}
		}

		day := strconv.Itoa(i)
		if len(day) == 1 {
			day = fmt.Sprintf("0%v", day)
		}
		monthStr := strconv.Itoa(month)
		if len(monthStr) == 1 {
			monthStr = fmt.Sprintf("0%v", monthStr)
		}

		btnText, data := textDayFunc(i, month, year)
		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, data)
		rowDays = append(rowDays, btn)
		amountWeek++
	}
	for i := 1; i <= 7-amountWeek; i++ {
		btn := tgbotapi.NewInlineKeyboardButtonData(" ", string(i))
		rowDays = append(rowDays, btn)
	}

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rowDays)

	return keyboard
}

func _date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func addSpecialButtons(
	year, month int, keyboard tgbotapi.InlineKeyboardMarkup, data TelegramCalendarSpecialButtonCallback, addPrev, addNext bool,
) tgbotapi.InlineKeyboardMarkup {
	var rowDays = []tgbotapi.InlineKeyboardButton{}
	if addPrev {
		prevMonth := month - 1
		prevYear := year
		if prevMonth < 1 {
			prevMonth = 12
			prevYear = year - 1
		}
		data.Month = fmt.Sprintf("%v.%v", prevYear, prevMonth)
		marshalData, err := json.Marshal(data)
		if err != nil {
			logrus.Error(err)
			return keyboard
		}
		btnPrev := tgbotapi.NewInlineKeyboardButtonData(BTN_PREV, string(marshalData))
		rowDays = append(rowDays, btnPrev)
	}
	if addNext {
		nextMonth := month + 1
		nextYear := year
		if nextMonth > 12 {
			nextMonth = 1
			nextYear = year + 1
		}
		data.Month = fmt.Sprintf("%v.%v", nextYear, nextMonth)
		marshalData, err := json.Marshal(data)
		if err != nil {
			logrus.Error(err)
			return keyboard
		}
		btnNext := tgbotapi.NewInlineKeyboardButtonData(BTN_NEXT, string(marshalData))
		rowDays = append(rowDays, btnNext)
	}
	if len(rowDays) > 0 {
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rowDays)
	}
	return keyboard
}
