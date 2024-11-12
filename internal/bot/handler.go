package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/AnVladic/DentalTelegramBot/internal/crm"
	"github.com/AnVladic/DentalTelegramBot/internal/database"
	"github.com/AnVladic/DentalTelegramBot/pkg"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

type TelegramBotHandler struct {
	bot             TelegramBotAPIWrapper
	userTexts       UserTexts
	dentalProClient crm.IDentalProClient
	db              *sql.DB
	branchID        int64
	location        *time.Location
	nowTime         TimeProvider
}

type HandlerMethod func(message *tgbotapi.Message, chatState *TelegramChatState)

func NewTelegramBotHandler(
	bot TelegramBotAPIWrapper,
	userTexts UserTexts,
	dentalProClient crm.IDentalProClient,
	db *sql.DB,
	branchID int64,
	location *time.Location,
	nowTime TimeProvider,
) *TelegramBotHandler {
	handler := &TelegramBotHandler{
		bot: bot, userTexts: userTexts, dentalProClient: dentalProClient, db: db, branchID: branchID,
		location: location, nowTime: nowTime,
	}
	return handler
}

func (h *TelegramBotHandler) StartCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	logrus.Print("/start command")
	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Welcome)

	user := database.User{
		ID:       -1,
		TgUserID: message.From.ID,
	}
	repository := database.UserRepository{DB: h.db}
	_, _, err := repository.GetOrCreateByTelegramID(user)
	if err != nil {
		logrus.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return
	}

	_, _ = h.Send(response, true)
}

func (h *TelegramBotHandler) RegisterCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	logrus.Print("/register command")
	log := logrus.WithFields(logrus.Fields{
		"module": "bot",
		"func":   "RegisterCommandHandler",
	})

	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Wait)
	newMsg, err := h.Send(response, true)
	if h.checkAndLogError(err, log, message, "") {
		return
	}
	h.ChangeToDoctorsMarkup(newMsg)
}

func (h *TelegramBotHandler) CancelCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	logrus.Print("/cancel command")
	chatState.UpdateChatState(nil)
	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Cancel)
	response.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	_, _ = h.Send(response, true)
}

func (h *TelegramBotHandler) UnknownCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	logrus.Print("/unknown command")
	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Welcome)
	_, _ = h.Send(response, true)
}

func (h *TelegramBotHandler) GetPhoneNumber(
	message *tgbotapi.Message, chatState *TelegramChatState) (bool, error) {
	if message.Contact == nil {
		text := "📲 Пожалуйста, нажмите кнопку <b>📞 Отправить номер телефона</b>, \n\n" +
			"Если передумали, введите команду /cancel ❌"
		response := tgbotapi.NewMessage(message.Chat.ID, text)
		response.ReplyMarkup = h.RequestContactKeyboard()
		response.ParseMode = "HTML"
		_, _ = h.Send(response, true)
		return false, nil
	}

	repository := database.UserRepository{DB: h.db}
	err := repository.UpsertContactByTelegramID(
		message.From.ID, message.Contact.FirstName, message.Contact.LastName, message.Contact.PhoneNumber,
	)
	if err != nil {
		logrus.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return false, err
	}
	return true, nil
}

func (h *TelegramBotHandler) NoAuthChangeNameHandler(
	message *tgbotapi.Message, chatState *TelegramChatState, onSuccess *HandlerMethod) {
	ok, err := h.GetPhoneNumber(message, chatState)
	if err != nil {
		_ = fmt.Errorf("GetPhoneNumber error %w", err)
		return
	}
	if !ok {
		chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
			h.NoAuthChangeNameHandler(message, chatState, onSuccess)
		})
		return
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.ContactsAddedSuccess)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	h.ChangeNameHandler(message, chatState, onSuccess)
}

func (h *TelegramBotHandler) ChangeNameHandler(
	message *tgbotapi.Message, chatState *TelegramChatState, onSuccess *HandlerMethod) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.handler",
		"func":   "ChangeNameHandler",
	})

	repository := database.UserRepository{DB: h.db}
	user, err := repository.GetUserByTelegramID(message.From.ID)
	if errors.Is(err, sql.ErrNoRows) || user.Phone == nil || *user.Phone == "" {
		h.RequestPhoneNumber(message)
		chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
			h.NoAuthChangeNameHandler(message, chatState, onSuccess)
		})
		return
	} else if h.checkAndLogError(err, log, message, "") {
		return
	}

	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.ChangeFirstNameRequest)
	_, _ = h.Send(response, true)

	chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
		h.ChangeFirstNameHandler(message, chatState, onSuccess)
	})
}

func (h *TelegramBotHandler) ChangeLastNameHandler(
	message *tgbotapi.Message, chatState *TelegramChatState, onSuccess *HandlerMethod) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.handler",
		"func":   "ChangeLastNameHandler",
	})

	repository := database.UserRepository{DB: h.db}
	err := repository.UpdateLastName(message.From.ID, message.Text)
	if h.checkAndLogError(err, log, message, "") {
		return
	}
	user, err := repository.GetUserByTelegramID(message.From.ID)
	if h.checkAndLogError(err, log, message, "") {
		return
	}

	patient, err := h.upsertCRMPatient(crm.Patient{
		Phone: *user.Phone, Name: *user.Name, Surname: *user.Lastname}, message, log)
	if err != nil {
		return
	}

	err = h.updateDentalProID(message.From.ID, patient.ExternalID, message, log)
	if err != nil {
		return
	}

	text := fmt.Sprintf(
		h.userTexts.ChangeNameSucceed, patient.Surname, patient.Name)
	response := tgbotapi.NewMessage(message.Chat.ID, text)
	response.ParseMode = "HTML"
	_, _ = h.Send(response, true)

	if onSuccess != nil {
		(*onSuccess)(message, chatState)
	}
}

func (h *TelegramBotHandler) ChangeFirstNameHandler(
	message *tgbotapi.Message, chatState *TelegramChatState, onSuccess *HandlerMethod) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.handler",
		"func":   "ChangeFirstNameHandler",
	})

	repository := database.UserRepository{DB: h.db}
	err := repository.UpdateFirstName(message.From.ID, message.Text)
	if h.checkAndLogError(err, log, message, "") {
		return
	}

	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.ChangeLastNameRequest)
	_, _ = h.Send(response, true)

	chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
		h.ChangeLastNameHandler(message, chatState, onSuccess)
	})
}

func (h *TelegramBotHandler) ShowRecordsListHandler(
	message *tgbotapi.Message, chatState *TelegramChatState) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.handler",
		"func":   "ShowRecordsListHandler",
	})

	user, err := h.findUserAndCheckPhoneNumber(
		h.ShowRecordsListHandler, chatState, message.From.ID, message, log)
	if err != nil {
		return
	}

	patient, _, err := h.getOrCreatePatient(*user.Name, *user.Lastname, *user.Phone, message, log)
	if err != nil {
		return
	}

	if user.DentalProID == nil {
		user.DentalProID = &patient.ExternalID
		if h.updateDentalProID(message.From.ID, patient.ExternalID, message, log) != nil {
			return
		}
	}

	records, err := h.getCRMRecordsList(patient.ExternalID, message, log)
	if err != nil {
		return
	}

	if len(records) == 0 {
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.HasNoRecords)
		_, _ = h.Send(response, true)
		return
	}

	rectorsTexts := make([]string, len(records))
	for i, record := range records {
		rectorsTexts[i] = fmt.Sprintf(h.userTexts.RecordItem,
			i+1,
			time.Time(record.DateStart).Format("2006-01-02 15:04"),
			record.DoctorName,
			record.DoctorGroup,
			record.Name,
			record.Duration,
		)
	}
	text := h.userTexts.RecordList + strings.Join(rectorsTexts, "\n\n")
	response := tgbotapi.NewMessage(message.Chat.ID, text)
	response.ParseMode = "HTML"
	_, _ = h.Send(response, true)
}

func (h *TelegramBotHandler) DeleteRecordHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.handler",
		"func":   "DeleteRecordHandler",
	})

	user, err := h.findUserAndCheckPhoneNumber(
		h.DeleteRecordHandler, chatState, message.From.ID, message, log)
	if err != nil {
		return
	}

	_, err = h.getDentalProIDByUser(user, message, log)
	if err != nil {
		return
	}

	records, err := h.getCRMRecordsList(*user.DentalProID, message, log)
	if err != nil {
		return
	}

	if len(records) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.HasNoRecords)
		_, _ = h.Send(msg, true)
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	keyboard.InlineKeyboard = make([][]tgbotapi.InlineKeyboardButton, len(records))
	for i, record := range records {
		datetime := time.Time(record.DateStart)
		keyboard.InlineKeyboard[i] = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf(
				h.userTexts.DeleteRecordItem, i+1, datetime.Format("2006-01-02 15:04"),
				record.DoctorName),
			fmt.Sprintf(`{"command":"del_r","r":%d}`, record.ID)),
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.DeleteRecords)
	msg.ReplyMarkup = keyboard
	_, _ = h.Send(msg, true)
}

func (h *TelegramBotHandler) ApproveRecordHandler(
	record crm.ShortRecord, message *tgbotapi.Message, chatState *TelegramChatState,
) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.handler",
		"func":   "ApproveRecordHandler",
	})

	var msg tgbotapi.MessageConfig

	datetime := time.Time(record.DateStart)

	if pkg.IsMatchIgnoreCase(message.Text, h.userTexts.NegativeAnswers) {
		text := fmt.Sprintf(h.userTexts.CancelDeleteRecord,
			datetime.Format("2006-01-02 15:04"),
			record.DoctorName,
		)
		msg = tgbotapi.NewMessage(message.Chat.ID, text)
	} else if pkg.IsMatchIgnoreCase(message.Text, h.userTexts.PositiveAnswers) {
		response, err := h.dentalProClient.DeleteRecord(record.ID)
		if !response.Status {
			err = &crm.RequestError{
				Code:    http.StatusNotFound,
				Message: response.Message,
			}
		}
		if h.checkAndLogError(err, log, message, "") {
			return
		}
		text := fmt.Sprintf(h.userTexts.SuccessDeleteRecord,
			datetime.Format("2006-01-02 15:04"),
			record.DoctorName,
		)
		msg = tgbotapi.NewMessage(message.Chat.ID, text)
	} else {
		msg = tgbotapi.NewMessage(message.Chat.ID, h.userTexts.UnknownApproveDeleteRecord)
		keyboard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("✅ Подтвердить"),
			tgbotapi.NewKeyboardButton("Отменить"),
		})
		keyboard.OneTimeKeyboard = true
		msg.ReplyMarkup = keyboard
		chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
			h.ApproveRecordHandler(record, message, chatState)
		})
	}
	_, _ = h.Send(msg, true)
}
