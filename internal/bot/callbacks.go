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
	"sort"
	"time"
)

type TelegramCalendarSpecialButtonCallback struct {
	CallbackData
	Month    string `json:"m"`
	DoctorID int64  `json:"d"`
}

type TelegramSpecialCallback struct {
	CallbackData
	Data string `json:"d"`
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

type TelegramRecordChangeCallback struct {
	CallbackData
	RecordID int64 `json:"r"`
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

	now := h.nowTime.Now()

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
			now := h.nowTime.Now()
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
			user.ID, query.Message.Chat.ID, query.Message.MessageID, callbackData.DoctorID,
			query.Message, log,
		)
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

	if user.DentalProID != nil {
		err := h.checkAndSendExistRecord(*user.DentalProID, callbackData.DoctorID, query.Message, log)
		if err != nil {
			return
		}
	}

	appointments, err := h.getAvailableAppointments(
		user, callbackData.DoctorID, query.Data, log, query.Message)
	if err != nil {
		return
	}

	keyboard := h.createAppointmentButtons(appointments, query, log)

	text := h.userTexts.ChooseAppointments
	if len(appointments) == 0 {
		text = h.noAppointmentsText(callbackData.DoctorID, query, log)
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID, query.Message.MessageID, text, keyboard)
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
		text = fmt.Sprintf(h.userTexts.DontHasIntervals, dataStr, doctor.FIO, appointment.Name, doctor.FIO)
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

func (h *TelegramBotHandler) UpdateNoAuthRegisterCommandHandler(query *tgbotapi.CallbackQuery, chatState *TelegramChatState) {
	chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
		h.NoAuthApproveRegister(query, message, chatState)
	})
}

func (h *TelegramBotHandler) NoAuthApproveRegister(
	query *tgbotapi.CallbackQuery, message *tgbotapi.Message, chatState *TelegramChatState) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.callback",
		"func":   "NoAuthApproveRegister",
	})
	ok, err := h.GetPhoneNumber(message, chatState)
	if err != nil {
		_ = fmt.Errorf("GetPhoneNumber error %w", err)
		return
	}
	if !ok {
		h.UpdateNoAuthRegisterCommandHandler(query, chatState)
		return
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.ContactsAddedSuccess)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	_, _ = h.Send(msg, true)

	newMessage, _ := h.Send(tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Wait), true)
	repository := database.UserRepository{DB: h.db}
	user, err := repository.GetUserByTelegramID(query.From.ID)
	if err != nil {
		return
	}

	register, err := h.getRegister(*user, query.Message, log)
	if err != nil {
		return
	}
	register.MessageID = newMessage.MessageID

	registerRepo := database.RegisterRepository{DB: h.db}
	err = registerRepo.Create(register)
	if h.checkAndLogError(err, log, message, "") {
		return
	}
	h.createApproveMessage(register, user, newMessage, log)
}

func (h *TelegramBotHandler) RegisterApproveCallback(
	query *tgbotapi.CallbackQuery, chatState *TelegramChatState) {
	var register *database.Register
	log := logrus.WithFields(logrus.Fields{
		"module": "callback",
		"func":   "RegisterApproveCallback",
	})

	parseData, err := h.parseTelegramChoiceIntervalCallback(query, log)
	if err != nil {
		return
	}

	repository := database.UserRepository{DB: h.db}
	user, err := repository.GetUserByTelegramID(query.From.ID)

	if user != nil {
		startTime, err := time.Parse("15:4", parseData.StartTime)
		if h.checkAndLogError(err, log, query.Message, "") {
			return
		}

		register, err = h.getRegister(*user, query.Message, log)
		if err != nil {
			return
		}
		datetime := time.Date(
			register.Datetime.Year(), register.Datetime.Month(), register.Datetime.Day(),
			startTime.Hour(), startTime.Minute(), 0, 0, h.location,
		)
		register.Datetime = &datetime
		err = h.updateRegisterDatetime(*register, query.Message, log)
		if err != nil {
			return
		}
	}

	if errors.Is(err, sql.ErrNoRows) || user == nil || user.Phone == nil || *user.Phone == "" {
		h.RequestPhoneNumber(query.Message)
		h.UpdateNoAuthRegisterCommandHandler(query, chatState)
		return
	} else if err != nil {
		log.Error(err)
		response := tgbotapi.NewMessage(query.Message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
	}

	h.createApproveMessage(register, user, query.Message, log)
}

func (h *TelegramBotHandler) RegisterCallback(query *tgbotapi.CallbackQuery) {
	log := logrus.WithFields(logrus.Fields{
		"module": "callback",
		"func":   "RegisterCallback",
	})

	user, err := h.getOrCreateUser(query.From.ID, query.Message, log)
	if err != nil {
		return
	}

	dentalProUser, _, err := h.getOrCreatePatient(*user.Name, *user.Lastname, *user.Phone, query.Message, log)
	if err != nil {
		return
	}

	if h.updateDentalProID(query.From.ID, dentalProUser.ExternalID, query.Message, log) != nil {
		return
	}

	register, err := h.getRegister(*user, query.Message, log)
	if err != nil {
		return
	}

	crmDoctor, err := h.getCRMDoctor(register.DoctorID, query.Message, log)
	if err != nil {
		return
	}

	err = h.checkAndSendExistRecord(dentalProUser.ExternalID, crmDoctor.ID, query.Message, log)
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
	chooseTime := pkg.DatetimeToTime(*register.Datetime)
	chooseDate := pkg.DatetimeToDate(*register.Datetime)
	if err != nil {
		return
	}
	for _, interval := range intervals {
		begin := time.Time(interval.Begin)
		if begin.Equal(chooseTime) {
			record, err := h.dentalProClient.RecordCreate(
				chooseDate, chooseTime,
				chooseTime.Add(time.Duration(appointment.Time)*time.Minute), *register.DoctorID,
				dentalProUser.ExternalID, appointment.ID, false,
			)
			if h.checkAndLogError(err, log, query.Message, "") {
				return
			}

			text := fmt.Sprintf(h.userTexts.RegisterSuccess,
				time.Time(record.Date).Format("2006-01-02"),
				time.Time(record.TimeBegin).Format("15:04:05"),
				crmDoctor.FIO,
				appointment.Name,
				appointment.Time,
				dentalProUser.Surname,
				dentalProUser.Name,
			)
			edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, text)
			edit.ParseMode = HTML
			_, _ = h.Edit(edit, true)
			return
		}
	}

	backKeyboard := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{h.getBackButton("calendar")}},
	}
	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID, query.Message.MessageID, h.userTexts.RegisterIntervalError, backKeyboard)
	_, _ = h.Edit(edit, true)
}

func (h *TelegramBotHandler) ChangeNameCallback(
	query *tgbotapi.CallbackQuery, chatState *TelegramChatState) {

	log := logrus.WithFields(logrus.Fields{
		"module": "callback",
		"func":   "RegisterCallback",
	})

	response := tgbotapi.NewMessage(query.Message.Chat.ID, h.userTexts.ChangeFirstNameRequest)
	_, _ = h.Send(response, true)

	handler := HandlerMethod(func(message *tgbotapi.Message, chatState *TelegramChatState) {
		newMessage, _ := h.Send(tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Wait), true)
		repository := database.UserRepository{DB: h.db}
		user, err := repository.GetUserByTelegramID(query.From.ID)
		if err != nil {
			return
		}

		register, err := h.getRegister(*user, query.Message, log)
		if err != nil {
			return
		}
		register.MessageID = newMessage.MessageID

		registerRepo := database.RegisterRepository{DB: h.db}
		err = registerRepo.Create(register)
		if h.checkAndLogError(err, log, message, "") {
			return
		}
		h.createApproveMessage(register, user, newMessage, log)
	})

	chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
		h.ChangeFirstNameHandler(message, chatState, &handler)
	})
}

func (h *TelegramBotHandler) ApproveDeleteRecord(
	query *tgbotapi.CallbackQuery, chatState *TelegramChatState) {
	log := logrus.WithFields(logrus.Fields{
		"module": "callback",
		"func":   "ApproveDeleteRecord",
	})
	recordData, err := h.parseTelegramRecordChangeCallback(query, log)
	if err != nil {
		return
	}

	user, err := h.findUserAndCheckPhoneNumber(
		func(message *tgbotapi.Message, chatState *TelegramChatState) {
			h.ApproveDeleteRecord(query, chatState)
		}, chatState, query.From.ID, query.Message, log,
	)
	if err != nil {
		return
	}

	_, err = h.getDentalProIDByUser(user, query.Message, log)
	if err != nil {
		return
	}

	records, err := h.getCRMRecordsList(*user.DentalProID, query.Message, log)
	if err != nil {
		return
	}

	for _, record := range records {
		if record.ID == recordData.RecordID {
			datetime := time.Time(record.DateStart)
			text := fmt.Sprintf(h.userTexts.ApproveDeleteRecord,
				datetime.Format("2006-01-02 15:04"), record.DoctorName)
			msg := tgbotapi.NewMessage(query.Message.Chat.ID, text)
			keyboard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{
				tgbotapi.NewKeyboardButton("✅ Подтвердить"),
				tgbotapi.NewKeyboardButton("Отменить"),
			})
			keyboard.OneTimeKeyboard = true
			msg.ReplyMarkup = keyboard
			_, _ = h.Send(msg, true)
			chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
				h.ApproveRecordHandler(record, message, chatState)
			})
			return
		}
	}

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, h.userTexts.HasNoDeleteRecord)
	_, _ = h.Send(msg, true)
}

func (h *TelegramBotHandler) sortRecords(records []crm.ShortRecord) {
	now := h.nowTime.Now().Unix()
	sort.Slice(records, func(i, j int) bool {
		afterNowI := records[i].DateStartTimestamp >= now
		afterNowJ := records[j].DateStartTimestamp >= now

		if afterNowI && !afterNowJ {
			return true
		}
		if !afterNowI && afterNowJ {
			return false
		}
		if afterNowI {
			return records[i].DateStartTimestamp < records[j].DateStartTimestamp
		}
		return records[i].DateStartTimestamp > records[j].DateStartTimestamp
	})
}
