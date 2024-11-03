package bot

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"time"
)

type TelegramCalendarSpecialButtonCallback struct {
	CallbackData
	Month    string `json:"m"`
	DoctorID int64  `json:"d"`
}

type TelegramSpecialCallback struct {
	CallbackData
	Data string
}

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
	Step int    `json:"s"`
}

type TelegramChoiceAppointmentCallback struct {
	CallbackData
	AppointmentID int64 `json:"a"`
}

type TelegramChoiceIntervalCallback struct {
	CallbackData
	StartTime string `json:"s"`
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

	appointment, err := h.getAppointment(
		user, register.DoctorID, register.AppointmentID, query.Data, log, query.Message)
	if err != nil {
		return
	}

	now := time.Now()

	text := fmt.Sprintf(
		"%s - %s\n%s\n🟢 Доступные дни", h.userTexts.Calendar, doctor.FIO, appointment.Name,
	)
	h.ChangeTimesheet(query, now, &text, doctor.ID, appointment.Time)
}

func (h *TelegramBotHandler) SwitchTimesheetMonthCallback(query *tgbotapi.CallbackQuery) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.callbacks",
		"func":   "SwitchTimesheetMonthCallback",
	})

	var specialButtonCallbackData TelegramCalendarSpecialButtonCallback
	var year, month int

	err := json.Unmarshal([]byte(query.Data), &specialButtonCallbackData)
	if h.checkAndLogError(err, log, query.Message, "SwitchTimesheetMonthCallback %s", err) {
		return
	}

	user, err := h.getOrCreateUser(query.From.ID, query.Message, log)
	if err != nil {
		return
	}

	register, err := h.getRegister(*user, query.Message, log)
	if err != nil {
		return
	}

	_, err = fmt.Sscanf(specialButtonCallbackData.Month, "%d.%d", &year, &month)
	if err != nil {
		if register.Datetime == nil {
			now := time.Now()
			register.Datetime = &now
		}
		year = register.Datetime.Year()
		month = int(register.Datetime.Month())
	}

	doctor, err := h.getDoctor(register.DoctorID, query.Message, log)
	if err != nil {
		return
	}

	appointment, err := h.getAppointment(
		user, register.DoctorID, register.AppointmentID, query.Data, log, query.Message)
	if err != nil {
		return
	}
	newDate := time.Date(
		year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	text := fmt.Sprintf(
		"%s - %s\n%s\n🟢 Доступные дни", h.userTexts.Calendar, doctor.FIO, appointment.Name,
	)
	h.ChangeTimesheet(query, newDate, &text, *register.DoctorID, appointment.Time)
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

	if callbackData.DoctorID > 0 {
		ok := h.upsertRegisterDoctorID(
			user.ID, query.Message.Chat.ID, query.Message.MessageID, callbackData.DoctorID, query.Message, log)
		if !ok {
			return
		}
	} else {
		register, err := h.getRegister(*user, query.Message, log)
		if err != nil {
			return
		}
		callbackData.DoctorID = *register.DoctorID
	}

	appointments, err := h.getAvailableAppointments(user, callbackData.DoctorID, query.Data, log, query.Message)
	if err != nil {
		return
	}

	keyboard := h.createAppointmentButtons(appointments, query, log)

	text := h.userTexts.ChooseAppointments
	if len(appointments) == 0 {
		text = h.noAppointmentsText(callbackData.DoctorID, query, log)
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, text, keyboard)
	_, _ = h.Edit(edit, true)
}

func (h *TelegramBotHandler) ChoiceDayCallback(query *tgbotapi.CallbackQuery) {
	log := logrus.WithFields(logrus.Fields{
		"module": "callback",
		"func":   "ChoiceDayCallback",
	})

	telegramChoiceDayCallback, err := h.parseTelegramChoiceDayCallbackData(query, log)
	if err != nil {
		return
	}

	user, err := h.getOrCreateUser(query.From.ID, query.Message, log)
	if err != nil {
		return
	}

	register, err := h.getRegister(*user, query.Message, log)
	if err != nil {
		return
	}

	doctor, err := h.getDoctor(register.DoctorID, query.Message, log)
	if err != nil {
		return
	}

	date, err := h.parseDate(telegramChoiceDayCallback.Date, query.Message, log)
	if err != nil {
		return
	}
	register.Datetime = &date

	appointment, err := h.getAppointment(
		user, &doctor.ID, register.AppointmentID, "", log, query.Message)
	if err != nil {
		return
	}

	intervals, err := h.getCRMFreeIntervals(register.DoctorID, date, appointment.Time, query.Message, log)
	if err != nil {
		return
	}

	err = h.updateRegisterDatetime(*register, query.Message, log)
	if err != nil {
		return
	}

	keyboard, err := h.createFreeIntervalsButtons(
		intervals, *telegramChoiceDayCallback, query.Message, log)
	if err != nil {
		return
	}

	var text string
	dataStr := date.Format("02.01.2006")
	if len(intervals) == 0 {
		text = fmt.Sprintf(h.userTexts.DontHasIntervals, dataStr, doctor.FIO, doctor.FIO, appointment.Name)
	} else {
		text = fmt.Sprintf(h.userTexts.ChooseInterval, dataStr, doctor.FIO, appointment.Name)
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID, query.Message.MessageID, text, keyboard)
	_, _ = h.Edit(edit, true)
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
	case "calendar":
		h.SwitchTimesheetMonthCallback(query)
	case "appointments":
		h.ShowAppointments(query)
	}
}

func (h *TelegramBotHandler) RegisterApproveCallback(query *tgbotapi.CallbackQuery) {
	log := logrus.WithFields(logrus.Fields{
		"module": "callback",
		"func":   "ChoiceDayCallback",
	})

	parseData, err := h.parseTelegramChoiceIntervalCallback(query, log)
	if err != nil {
		return
	}
	chooseTime, err := time.Parse("15:04", parseData.StartTime)
	if h.checkAndLogError(
		err, log, query.Message, "parseData.StartTime does not parse %s", parseData.StartTime) {
		return
	}

	user, err := h.getOrCreateUser(query.From.ID, query.Message, log)
	if err != nil {
		return
	}

	register, err := h.getRegister(*user, query.Message, log)
	if err != nil {
		return
	}

	appointment, err := h.getAppointment(
		user, register.DoctorID, register.AppointmentID, "", log, query.Message)
	if err != nil {
		return
	}

	intervals, err := h.getCRMFreeIntervals(
		register.DoctorID, *register.Datetime, appointment.Time, query.Message, log,
	)
	if err != nil {
		return
	}
	for _, interval := range intervals {
		if time.Time(interval.Begin).Equal(chooseTime) {
			//edit := tgbotapi.NewEditMessageTextAndMarkup(
			//	query.Message.Chat.ID, query.Message.MessageID, text, keyboard)
			//return
		}
	}

}
