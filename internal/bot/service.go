package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AnVladic/DentalTelegramBot/internal/crm"
	"github.com/AnVladic/DentalTelegramBot/internal/database"
	"github.com/AnVladic/DentalTelegramBot/pkg"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const HTML = "HTML"

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

func findScheduleByDate(day, month, year int, schedule []crm.DayInterval) *crm.DayInterval {
	for _, entry := range schedule {
		date := time.Time(entry.Date)
		if date.Day() == day &&
			date.Month() == time.Month(month) &&
			date.Year() == year {
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
	msg.ParseMode = HTML
	_, _ = h.Send(msg, true)
}

func (h *TelegramBotHandler) getBackButton(back string) tgbotapi.InlineKeyboardButton {
	data := TelegramBackCallback{
		CallbackData{"back"},
		back,
	}
	marshalData, _ := json.Marshal(data)
	btn := tgbotapi.NewInlineKeyboardButtonData(h.userTexts.Back, string(marshalData))
	return btn
}

func (h *TelegramBotHandler) AddBackButton(
	keyboard tgbotapi.InlineKeyboardMarkup, back string) tgbotapi.InlineKeyboardMarkup {
	btn := h.getBackButton(back)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tgbotapi.InlineKeyboardButton{btn})
	return keyboard
}

func (h *TelegramBotHandler) GenerateTimesheetCalendar(
	schedule []crm.DayInterval, currentDate time.Time, doctorID int64) tgbotapi.InlineKeyboardMarkup {
	textDayFunc := func(day, month, year int) (string, string) {
		btnText := fmt.Sprintf("%v", day)
		now := h.nowTime.Now()
		if now.Day() <= day || int(now.Month()) < month || now.Year() < year {
			workSchedule := findScheduleByDate(day, month, year, schedule)
			if workSchedule != nil && len(workSchedule.Slots) > 0 {
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

	now := h.nowTime.Now()
	year := currentDate.Year()
	month := currentDate.Month()
	showPrev := now.Year() < year || now.Month() < month
	keyboard := tgbotapi.InlineKeyboardMarkup{}
	keyboard = addMonthYearRow(year, month, keyboard)
	keyboard = addDaysNamesRow(keyboard)
	keyboard = h.generateMonth(year, int(month), keyboard, textDayFunc)
	keyboard = addSpecialButtons(year, int(month), keyboard, specialButtonCallbackData, showPrev,
		currentDate.Sub(now) < 365*24*time.Hour)
	keyboard = h.AddBackButton(keyboard, "appointments")
	return keyboard
}

func (h *TelegramBotHandler) ChangeTimesheet(
	query *tgbotapi.CallbackQuery, start time.Time, text *string, doctorID int64, duration int,
) {
	nextMonth := start.AddDate(0, 1, -start.Day()+1)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	schedule, err := h.dentalProClient.FreeIntervals(
		start, nextMonth, -1, doctorID, h.branchID, duration,
	)
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

func (h *TelegramBotHandler) CheckDoctorBranch(doctor crm.Doctor, branchID int64) bool {
	branchStr := strconv.FormatInt(branchID, 10)
	for branch := range doctor.Branches {
		if branch == branchStr {
			return true
		}
	}
	return false
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
		if !h.CheckDoctorBranch(doctor, h.branchID) {
			continue
		}

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
			"%s - %s", doctor.FIO, strings.Join(pkg.GetMapValues(doctor.Departments), ", "))
		btn := tgbotapi.NewInlineKeyboardButtonData(title, string(bytesData))
		row := []tgbotapi.InlineKeyboardButton{btn}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}
	response := tgbotapi.NewEditMessageTextAndMarkup(message.Chat.ID, message.MessageID,
		h.userTexts.ChooseDoctor,
		keyboard)
	_, _ = h.Edit(response, true)
}

// getOrCreatePatient return: Patient, created, error
func (h *TelegramBotHandler) getOrCreatePatient(name, surname, phone string,
	message *tgbotapi.Message, log *logrus.Entry) (crm.Patient, bool, error) {
	patient, err := h.dentalProClient.PatientByPhone(phone)
	var reqErr *crm.RequestError
	if errors.As(err, &reqErr) && reqErr.Code == http.StatusNotFound {
		patient, err = h.dentalProClient.CreatePatient(name, surname, phone)
		if h.checkAndLogError(err, log, message, "") {
			return crm.Patient{}, false, err
		}
		return patient, true, nil
	}
	if h.checkAndLogError(err, log, message, "") {
		return patient, false, err
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

func (h *TelegramBotHandler) getAppointment(
	user *database.User, doctorID, appointmentID *int64, data string,
	log *logrus.Entry, message *tgbotapi.Message,
) (*crm.Appointment, error) {
	if doctorID == nil || appointmentID == nil {
		err := fmt.Errorf("doctorID=%d or appointmentID=%d is nil", doctorID, appointmentID)
		h.checkAndLogError(err, log, message, "")
		return nil, err
	}

	appointments, err := h.getAvailableAppointments(user, *doctorID, data, log, message)
	if err != nil {
		return nil, err
	}
	for _, appointmentsList := range appointments {
		for _, appointment := range appointmentsList {
			if appointment.ID == *appointmentID {
				return &appointment, err
			}
		}
	}
	err = fmt.Errorf("appointment Not Found")
	h.checkAndLogError(err, log, message, "Appointment does not found, %d", appointmentID)
	return nil, err
}

func (h *TelegramBotHandler) createAppointmentButtons(
	appointments map[int64]map[int64]crm.Appointment,
	query *tgbotapi.CallbackQuery, log *logrus.Entry,
) tgbotapi.InlineKeyboardMarkup {
	var allAppointments []crm.Appointment

	for _, doctorAppointments := range appointments {
		for _, appointment := range doctorAppointments {
			allAppointments = append(allAppointments, appointment)
		}
	}

	sort.Slice(allAppointments, func(i, j int) bool {
		return allAppointments[i].Time < allAppointments[j].Time
	})

	buttons := make([][]tgbotapi.InlineKeyboardButton, 0)
	for _, appointment := range allAppointments {
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

func (h *TelegramBotHandler) parseTelegramChoiceIntervalCallback(
	query *tgbotapi.CallbackQuery, log *logrus.Entry) (*TelegramChoiceIntervalCallback, error) {
	var telegramChoiceIntervalCallback TelegramChoiceIntervalCallback
	err := json.Unmarshal([]byte(query.Data), &telegramChoiceIntervalCallback)
	if h.checkAndLogError(err, log, query.Message, "TelegramChoiceIntervalCallback Unmarshal error") {
		log.Error(err)
		return nil, err
	}
	return &telegramChoiceIntervalCallback, nil
}

func (h *TelegramBotHandler) parseTelegramRecordChangeCallback(
	query *tgbotapi.CallbackQuery, log *logrus.Entry) (*TelegramRecordChangeCallback, error) {
	var telegramRecordChangeCallback TelegramRecordChangeCallback
	err := json.Unmarshal([]byte(query.Data), &telegramRecordChangeCallback)
	if h.checkAndLogError(err, log, query.Message, "TelegramRecordChangeCallback Unmarshal error") {
		log.Error(err)
		return nil, err
	}
	return &telegramRecordChangeCallback, nil
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

func (h *TelegramBotHandler) getCRMPatient(
	phoneNumber string, message *tgbotapi.Message, log *logrus.Entry) (*crm.Patient, error) {
	patient, err := h.dentalProClient.PatientByPhone(phoneNumber)
	var reqErr *crm.RequestError
	if errors.As(err, &reqErr) {
		if reqErr.Code == http.StatusNotFound {
			return nil, err
		}
	}

	if h.checkAndLogError(err, log, message, "PatientByPhone %s", phoneNumber) {
		return &patient, err
	}
	return &patient, nil
}

func (h *TelegramBotHandler) upsertCRMPatient(
	patient crm.Patient, message *tgbotapi.Message, log *logrus.Entry) (*crm.Patient, error) {
	dentalProUser, err := h.getCRMPatient(patient.Phone, message, log)
	var reqErr *crm.RequestError
	if errors.As(err, &reqErr) {
		if reqErr.Code == http.StatusNotFound {
			newPatient, err := h.dentalProClient.CreatePatient(
				patient.Name, patient.Surname, patient.Phone,
			)
			if h.checkAndLogError(err, log, message, "CreatePatient %s", patient.Phone) {
				return nil, err
			}
			return &newPatient, nil
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	patient.ExternalID = dentalProUser.ExternalID
	status, err := h.dentalProClient.EditPatient(patient)
	if err != nil || !status.Status {
		if err == nil {
			err = fmt.Errorf("EditPatient error %s", status.Message)
		}
		h.checkAndLogError(err, log, message, "EditPatient %s", patient.Phone)
		return nil, err
	}
	return &patient, err
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

	date = ToDate(date)
	freeIntervals, err := h.dentalProClient.FreeIntervals(
		date, date, -1, *doctorID, h.branchID, duration)
	if len(freeIntervals) == 0 {
		return []crm.TimeRange{}, err
	}
	times := freeIntervals[0].Slots[0].Time
	sort.Slice(times, func(i, j int) bool {
		return time.Time(times[i].Begin).Before(time.Time(times[j].Begin))
	})
	if err != nil {
		log.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return nil, err
	}
	return times, nil
}

func ToDate(datetime time.Time) time.Time {
	return time.Date(
		datetime.Year(), datetime.Month(), datetime.Day(), 0, 0, 0, 0, datetime.Location())
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

func (h *TelegramBotHandler) localTimeCutoff() time.Time {
	return h.nowTime.Now().Add(15 * time.Minute).In(h.location)
}

func (h *TelegramBotHandler) generateIntervalButtons(
	intervals []crm.TimeRange, date time.Time) ([][]tgbotapi.InlineKeyboardButton, error) {
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0)
	rowLen := 3
	cutoff := h.localTimeCutoff()

	for i, interval := range intervals {
		if i%rowLen == 0 {
			buttons = append(buttons, make([]tgbotapi.InlineKeyboardButton, 0, rowLen))
		}
		timeOfDay := time.Time(interval.Begin)
		combined := time.Date(
			date.Year(), date.Month(), date.Day(),
			timeOfDay.Hour(), timeOfDay.Minute(), 0, 0, h.location,
		)
		beginStr := timeOfDay.Format("15:04")
		endStr := time.Time(interval.End).Format("15:04")
		data, err := json.Marshal(TelegramChoiceIntervalCallback{
			CallbackData{"interval"},
			beginStr,
		})
		if err != nil {
			return nil, err
		}
		text := fmt.Sprintf("%s - %s", beginStr, endStr)
		if combined.Before(cutoff) {
			text = "❌ " + text
		}
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
	date, err := time.Parse("2006.1.2", choiceData.Date)
	if h.checkAndLogError(err, log, message, "Date format is invalid") {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}
	intervalSubset := h.paginateIntervals(intervals, choiceData, maxIntervalsCount)
	intervalButtons, err := h.generateIntervalButtons(intervalSubset, date)
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

func (h *TelegramBotHandler) updateRegisterDatetime(
	register database.Register,
	message *tgbotapi.Message,
	log *logrus.Entry,
) error {
	registerRepo := database.RegisterRepository{DB: h.db}
	err := registerRepo.UpdateDatetime(register)
	if h.checkAndLogError(err, log, message, "Update Register err") {
		return err
	}
	return nil
}

func (h *TelegramBotHandler) createApproveRegisterKeyboard() tgbotapi.InlineKeyboardMarkup {
	approveData := TelegramSpecialCallback{
		CallbackData{"approve"},
		"register",
	}
	approveDataBytes, _ := json.Marshal(approveData)

	changeNameData := TelegramSpecialCallback{
		CallbackData{"change_name"},
		"register",
	}
	changeNameDataBytes, _ := json.Marshal(changeNameData)

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData(h.userTexts.ChangeName, string(changeNameDataBytes)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(h.userTexts.Approve, string(approveDataBytes)),
			h.getBackButton("calendar"),
		},
	}}
}

func (h *TelegramBotHandler) createApproveMessage(
	register *database.Register,
	user *database.User,
	message *tgbotapi.Message,
	log *logrus.Entry,
) {
	dentalProUser, err := h.getCRMPatient(*user.Phone, message, log)
	var reqErr *crm.RequestError
	if errors.As(err, &reqErr) && reqErr.Code != http.StatusNotFound {
		return
	}

	selfUser := SelfUser{user, dentalProUser}

	appointment, err := h.getAppointment(
		user, register.DoctorID, register.AppointmentID, "", log, message)
	if err != nil {
		return
	}

	doctor, err := h.getDoctor(register.DoctorID, message, log)
	if err != nil {
		return
	}

	if h.localTimeCutoff().After(*register.Datetime) {
		backKeyboard := tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{h.getBackButton("calendar")}},
		}
		edit := tgbotapi.NewEditMessageTextAndMarkup(
			message.Chat.ID, message.MessageID, h.userTexts.ApproveRegisterTimeLimit, backKeyboard,
		)
		_, _ = h.Edit(edit, true)
	} else {
		text := fmt.Sprintf(
			h.userTexts.ApproveRegister,
			register.Datetime.Format("2006-01-02 15:04"),
			doctor.FIO,
			appointment.Name,
			appointment.Time,
			selfUser.GetSelfLastName(),
			selfUser.GetSelfFirstName(),
		)
		edit := tgbotapi.NewEditMessageTextAndMarkup(
			message.Chat.ID, message.MessageID, text, h.createApproveRegisterKeyboard())
		edit.ParseMode = HTML
		_, _ = h.Edit(edit, true)
	}
}

func (h *TelegramBotHandler) getCRMRecordsList(crmUserID int64,
	message *tgbotapi.Message,
	log *logrus.Entry) ([]crm.ShortRecord, error) {
	records, err := h.dentalProClient.PatientRecords(crmUserID)
	if h.checkAndLogError(err, log, message, "Get CRM Record List err") {
		return nil, err
	}
	return records, nil
}

func (h *TelegramBotHandler) findRecordByPatientAndDoctor(
	doctorID, patientID int64, message *tgbotapi.Message, log *logrus.Entry) (*crm.ShortRecord, error) {
	records, err := h.dentalProClient.PatientRecords(patientID)
	if h.checkAndLogError(err, log, message, "PatientRecords %s", err) {
		return nil, err
	}
	for _, record := range records {
		if record.DoctorID == doctorID {
			return &record, nil
		}
	}
	return nil, nil
}

func (h *TelegramBotHandler) updateDentalProID(
	telegramID, dentalProID int64, message *tgbotapi.Message, log *logrus.Entry) error {
	userRepo := database.UserRepository{DB: h.db}
	err := userRepo.UpdateDentalProIDByTelegramID(telegramID, dentalProID)
	if h.checkAndLogError(
		err, log, message, "updateDentalProID tg=%d dentalPro=%d", telegramID, dentalProID) {
		return err
	}
	return nil
}

// Запрашивает у юзера номер телефона
func (h *TelegramBotHandler) noAuthRequest(
	successFunc func(message *tgbotapi.Message, chatState *TelegramChatState), chatState *TelegramChatState,
	message *tgbotapi.Message) error {

	ok, err := h.GetPhoneNumber(message, chatState)
	if err != nil {
		_ = fmt.Errorf("GetPhoneNumber error %w", err)
		return err
	}
	if !ok {
		chatState.UpdateChatState(func(message *tgbotapi.Message, lChatState *TelegramChatState) {
			_ = h.noAuthRequest(successFunc, lChatState, message)
		})
		return nil
	}
	successFunc(message, chatState)
	return nil
}

func (h *TelegramBotHandler) findUserAndCheckPhoneNumber(
	successFunc func(message *tgbotapi.Message, chatState *TelegramChatState), chatState *TelegramChatState,
	fromID int64,
	message *tgbotapi.Message, log *logrus.Entry,
) (*database.User, error) {
	repository := database.UserRepository{DB: h.db}
	user, err := repository.GetUserByTelegramID(fromID)
	if errors.Is(err, sql.ErrNoRows) || user.Phone == nil || *user.Phone == "" {
		if err == nil {
			err = fmt.Errorf("user.Phone is empty")
		}
		h.RequestPhoneNumber(message)
		chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
			_ = h.noAuthRequest(successFunc, chatState, message)
		})
		return nil, err
	} else if h.checkAndLogError(err, log, message, "") {
		return nil, err
	}
	return user, nil
}

func (h *TelegramBotHandler) getDentalProIDByUser(
	user *database.User, message *tgbotapi.Message, log *logrus.Entry,
) (int64, error) {
	if user.DentalProID == nil {
		patient, _, err := h.getOrCreatePatient(*user.Name, *user.Lastname, *user.Phone, message, log)
		if h.checkAndLogError(err, log, message, "") {
			return 0, err
		}
		user.DentalProID = &patient.ExternalID
		err = h.updateDentalProID(message.From.ID, patient.ExternalID, message, log)
		if err != nil {
			return 0, err
		}
	}
	return *user.DentalProID, nil
}
