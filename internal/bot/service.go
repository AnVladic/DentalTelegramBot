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
	"sort"
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
			fmt.Sprintf("%v.%v.%v", year, month, day),
			0,
		}
		dataBytes, _ := json.Marshal(data)
		return btnText, string(dataBytes)
	}

	specialButtonCallbackData := TelegramCalendarSpecialButtonCallback{
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

func (h *TelegramBotHandler) parseDoctorCallbackData(
	query *tgbotapi.CallbackQuery) (TelegramBotDoctorCallbackData, error) {
	var data TelegramBotDoctorCallbackData
	err := json.Unmarshal([]byte(query.Data), &data)
	return data, err
}

func (h *TelegramBotHandler) getOrCreateUser(
	tgUserID int64, message *tgbotapi.Message, log *logrus.Entry) (*database.User, error) {
	repository := database.UserRepository{DB: h.db}
	user, _, err := repository.GetOrCreateByTelegramID(database.User{TgUserID: tgUserID})
	if h.checkAndLogError(err, log, message, "GetOrCreateByTelegramID Unknown error") {
		return nil, err
	}
	return user, nil
}

func (h *TelegramBotHandler) upsertRegisterDoctorID(
	userID int64, chatID int64, messageID int, doctorID int64, message *tgbotapi.Message, log *logrus.Entry,
) bool {
	registerRepo := database.RegisterRepository{DB: h.db}
	_, err := registerRepo.UpsertDoctorID(database.Register{
		UserID:    userID,
		ChatID:    chatID,
		MessageID: messageID,
		DoctorID:  &doctorID,
	})
	return !h.checkAndLogError(err, log, message, "UpsertDoctorID %d", doctorID)
}

func (h *TelegramBotHandler) getAvailableAppointments(
	user *database.User, doctorID int64, data string, log *logrus.Entry, message *tgbotapi.Message,
) (map[int64]map[int64]crm.Appointment, error) {
	var clientID int64 = 1
	if user.DentalProID != nil && *user.DentalProID > 0 {
		clientID = *user.DentalProID
	}
	appointments, err := h.dentalProClient.AvailableAppointments(clientID, []int64{doctorID}, false)
	if h.checkAndLogError(err, log, message, "Get Appointments error, %s", data) {
		return nil, err
	}
	return appointments, nil
}

func (h *TelegramBotHandler) createAppointmentButtons(
	appointments map[int64]map[int64]crm.Appointment,
	query *tgbotapi.CallbackQuery, log *logrus.Entry,
) tgbotapi.InlineKeyboardMarkup {
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0)
	for _, doctorAppointments := range appointments {
		for _, appointment := range doctorAppointments {
			data, err := json.Marshal(TelegramChoiceAppointmentCallback{
				CallbackData{"appointment"},
				appointment.ID,
			})
			if h.checkAndLogError(err, log, query.Message, "Marshal error") {
				continue
			}

			text := fmt.Sprintf("(%d мин.) %s", appointment.Time, appointment.Name)
			button := tgbotapi.NewInlineKeyboardButtonData(text, string(data))
			buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
		}
	}
	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
	return h.AddBackButton(keyboard, "doctors")
}

func (h *TelegramBotHandler) noAppointmentsText(doctorID int64, query *tgbotapi.CallbackQuery, log *logrus.Entry) string {
	doctorRepo := database.DoctorRepository{DB: h.db}
	doctor, err := doctorRepo.Get(doctorID)
	if h.checkAndLogError(err, log, query.Message, "Get Doctor ByID error, %s", query.Data) {
		return ""
	}
	return fmt.Sprintf(h.userTexts.DontHasAppointments, doctor.FIO)
}

func (h *TelegramBotHandler) parseChoiceAppointmentCallbackData(
	query *tgbotapi.CallbackQuery, log *logrus.Entry) (*TelegramChoiceAppointmentCallback, error) {
	var telegramChoiceAppointmentCallback TelegramChoiceAppointmentCallback
	err := json.Unmarshal([]byte(query.Data), &telegramChoiceAppointmentCallback)
	if h.checkAndLogError(err, log, query.Message, "TelegramChoiceAppointmentCallback Unmarshal error") {
		log.Error(err)
		return nil, err
	}
	return &telegramChoiceAppointmentCallback, nil
}

func (h *TelegramBotHandler) parseTelegramBotDoctorCallbackData(
	query *tgbotapi.CallbackQuery, log *logrus.Entry) (*TelegramBotDoctorCallbackData, error) {
	var telegramBotDoctorCallbackData TelegramBotDoctorCallbackData
	err := json.Unmarshal([]byte(query.Data), &telegramBotDoctorCallbackData)
	if h.checkAndLogError(err, log, query.Message, "TelegramBotDoctorCallbackData Unmarshal error") {
		log.Error(err)
		return nil, err
	}
	return &telegramBotDoctorCallbackData, nil
}

func (h *TelegramBotHandler) parseTelegramChoiceDayCallbackData(
	query *tgbotapi.CallbackQuery, log *logrus.Entry) (*TelegramChoiceDayCallback, error) {
	var telegramChoiceDayCallback TelegramChoiceDayCallback
	err := json.Unmarshal([]byte(query.Data), &telegramChoiceDayCallback)
	if h.checkAndLogError(err, log, query.Message, "TelegramChoiceDayCallback Unmarshal error") {
		log.Error(err)
		return nil, err
	}
	return &telegramChoiceDayCallback, nil
}

func (h *TelegramBotHandler) updateAppointmentRegister(
	user database.User, message *tgbotapi.Message, appointmentID int64, log *logrus.Entry,
) error {
	registerRepo := database.RegisterRepository{DB: h.db}
	err := registerRepo.UpdateAppointmentID(database.Register{
		UserID:        user.ID,
		ChatID:        message.Chat.ID,
		MessageID:     message.MessageID,
		AppointmentID: &appointmentID,
	})
	if h.checkAndLogError(err, log, message, "UpdateAppointmentID %d", appointmentID) {
		return err
	}
	return nil
}

func (h *TelegramBotHandler) getRegister(
	user database.User, message *tgbotapi.Message, log *logrus.Entry,
) (*database.Register, error) {
	registerRepo := database.RegisterRepository{DB: h.db}
	register, err := registerRepo.Get(user.ID, message.Chat.ID, message.MessageID)
	if h.checkAndLogError(
		err, log, message, "Get Register by %d, %d, %d", user.ID, message.Chat.ID, message.MessageID) {
		return nil, err
	}
	return register, nil
}

func (h *TelegramBotHandler) getCRMDoctor(
	doctorID *int64, message *tgbotapi.Message, log *logrus.Entry) (*crm.Doctor, error) {
	doctors, err := h.dentalProClient.DoctorsList()
	if err != nil {
		log.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return nil, err
	}

	if doctorID == nil {
		err := fmt.Errorf("doctor ID is nil")
		log.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return nil, err
	}

	doctor := crm.GetDoctorByID(doctors, *doctorID)
	if doctor == nil {
		log.Error("Doctor doest found")
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
	}
	return doctor, nil
}

func (h *TelegramBotHandler) getDoctor(
	doctorID *int64, message *tgbotapi.Message, log *logrus.Entry) (*database.Doctor, error) {
	if doctorID == nil {
		err := fmt.Errorf("doctor ID is nil")
		h.checkAndLogError(err, log, message, "doctor ID is nil")
		return nil, err
	}

	doctorRepo := database.DoctorRepository{DB: h.db}
	doctor, err := doctorRepo.Get(*doctorID)
	if h.checkAndLogError(err, log, message, "Get Doctor By %d", doctorID) {
		return nil, err
	}
	return doctor, nil
}

func (h *TelegramBotHandler) parseDate(
	dateStr string, message *tgbotapi.Message, log *logrus.Entry) (time.Time, error) {
	date, err := time.Parse("2006.1.2", dateStr)
	if h.checkAndLogError(err, log, message, "date str is %s", dateStr) {
		return date, err
	}
	return date, nil
}

func (h *TelegramBotHandler) getCRMFreeIntervals(
	doctorID *int64, date time.Time, duration int,
	message *tgbotapi.Message, log *logrus.Entry) ([]crm.TimeRange, error) {
	if doctorID == nil {
		err := fmt.Errorf("doctor ID is nil")
		log.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return nil, err
	}

	freeIntervals, err := h.dentalProClient.FreeIntervals(*doctorID, date, duration)
	sort.Slice(freeIntervals, func(i, j int) bool {
		return freeIntervals[i].Begin.Before(freeIntervals[j].Begin)
	})
	if err != nil {
		log.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return nil, err
	}
	return freeIntervals, nil
}

func (h *TelegramBotHandler) createSpacialButton(text, data string) tgbotapi.InlineKeyboardButton {
	dataBytes, _ := json.Marshal(TelegramSpecialCallback{
		CallbackData{"spacial"},
		data,
	})
	button := tgbotapi.NewInlineKeyboardButtonData(text, string(dataBytes))
	return button
}

func (h *TelegramBotHandler) createDataString(
	data interface{}, message *tgbotapi.Message, log *logrus.Entry) (string, error) {
	dataBytes, err := json.Marshal(data)
	if h.checkAndLogError(err, log, message, "Marshal json") {
		return "", err
	}
	return string(dataBytes), nil
}

func (h *TelegramBotHandler) paginateIntervals(
	intervals []crm.TimeRange, choiceData TelegramChoiceDayCallback, maxIntervalsCount int) []crm.TimeRange {
	start := maxIntervalsCount * choiceData.Step
	end := start + maxIntervalsCount
	if end > len(intervals) {
		end = len(intervals)
	}
	if start > len(intervals) {
		start = len(intervals)
	}
	return intervals[start:end]
}

func (h *TelegramBotHandler) generateIntervalButtons(
	intervals []crm.TimeRange) ([][]tgbotapi.InlineKeyboardButton, error) {
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0)
	rowLen := 3
	for i, interval := range intervals {
		if i%rowLen == 0 {
			buttons = append(buttons, make([]tgbotapi.InlineKeyboardButton, 0, rowLen))
		}
		beginStr := interval.Begin.Format("15:04")
		endStr := interval.End.Format("15:04")
		data, err := json.Marshal(TelegramChoiceIntervalCallback{
			CallbackData{"interval"},
			beginStr,
		})
		if err != nil {
			return nil, err
		}
		text := fmt.Sprintf("%s - %s", beginStr, endStr)
		button := tgbotapi.NewInlineKeyboardButtonData(text, string(data))
		buttons[len(buttons)-1] = append(buttons[len(buttons)-1], button)
	}
	return buttons, nil
}

func (h *TelegramBotHandler) createNavigationButtons(
	intervals []crm.TimeRange,
	choiceData TelegramChoiceDayCallback,
	maxIntervalsCount int,
	message *tgbotapi.Message,
	log *logrus.Entry,
) ([]tgbotapi.InlineKeyboardButton, error) {
	var specialButtons []tgbotapi.InlineKeyboardButton
	if choiceData.Step > 0 {
		prevChoiceData := choiceData
		prevChoiceData.Step--
		dataStr, err := h.createDataString(prevChoiceData, message, log)
		if err != nil {
			return nil, err
		}
		specialButtons = append(specialButtons, tgbotapi.NewInlineKeyboardButtonData(BTN_PREV, dataStr))
	}
	if len(intervals) > maxIntervalsCount*(choiceData.Step+1) {
		nextChoiceData := choiceData
		nextChoiceData.Step++
		dataStr, err := h.createDataString(nextChoiceData, message, log)
		if err != nil {
			return nil, err
		}
		specialButtons = append(specialButtons, tgbotapi.NewInlineKeyboardButtonData(BTN_NEXT, dataStr))
	}
	return specialButtons, nil
}

func (h *TelegramBotHandler) createFreeIntervalsButtons(
	intervals []crm.TimeRange,
	choiceData TelegramChoiceDayCallback,
	message *tgbotapi.Message,
	log *logrus.Entry,
) (tgbotapi.InlineKeyboardMarkup, error) {
	maxIntervalsCount := 24
	intervalSubset := h.paginateIntervals(intervals, choiceData, maxIntervalsCount)
	intervalButtons, err := h.generateIntervalButtons(intervalSubset)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: intervalButtons}

	if len(intervals) > maxIntervalsCount {
		navigationButtons, err := h.createNavigationButtons(
			intervals, choiceData, maxIntervalsCount, message, log)
		if err != nil {
			return tgbotapi.InlineKeyboardMarkup{}, err
		}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, navigationButtons)
	}

	return h.AddBackButton(keyboard, "calendar"), nil
}
