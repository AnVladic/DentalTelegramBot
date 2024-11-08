package bot

import (
	"database/sql"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
	"main/internal/database"
	"strings"
	"time"
)

type TelegramBotHandler struct {
	bot             *tgbotapi.BotAPI
	userTexts       UserTexts
	dentalProClient crm.IDentalProClient
	db              *sql.DB
	branchID        int64
	location        *time.Location
}

type HandlerMethod func(message *tgbotapi.Message, chatState *TelegramChatState)

func NewTelegramBotHandler(
	bot *tgbotapi.BotAPI,
	userTexts UserTexts,
	dentalProClient crm.IDentalProClient,
	db *sql.DB,
	branchID int64,
	location *time.Location,
) *TelegramBotHandler {
	handler := &TelegramBotHandler{
		bot: bot, userTexts: userTexts, dentalProClient: dentalProClient, db: db, branchID: branchID,
		location: location,
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

	go func() {
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Wait)
		newMsg, err := h.Send(response, true)
		if h.checkAndLogError(err, log, message, "") {
			return
		}
		h.ChangeToDoctorsMarkup(newMsg)
	}()
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
	chatState.UpdateChatState(func(message *tgbotapi.Message, chatState *TelegramChatState) {
		h.ChangeNameHandler(message, chatState, onSuccess)
	})
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

func (h *TelegramBotHandler) NoAuthShowRecordsListHandler(
	message *tgbotapi.Message, chatState *TelegramChatState) {
	ok, err := h.GetPhoneNumber(message, chatState)
	if err != nil {
		_ = fmt.Errorf("GetPhoneNumber error %w", err)
		return
	}
	if !ok {
		chatState.UpdateChatState(h.NoAuthShowRecordsListHandler)
		return
	}
	h.ShowRecordsListHandler(message, chatState)
}

func (h *TelegramBotHandler) ShowRecordsListHandler(
	message *tgbotapi.Message, chatState *TelegramChatState) {
	log := logrus.WithFields(logrus.Fields{
		"module": "bot.handler",
		"func":   "ShowRecordsList",
	})

	repository := database.UserRepository{DB: h.db}
	user, err := repository.GetUserByTelegramID(message.From.ID)
	if errors.Is(err, sql.ErrNoRows) || user.Phone == nil || *user.Phone == "" {
		h.RequestPhoneNumber(message)
		chatState.UpdateChatState(h.NoAuthShowRecordsListHandler)
		return
	} else if h.checkAndLogError(err, log, message, "") {
		return
	}

	patient, _, err := h.getOrCreatePatient(*user.Name, *user.Lastname, *user.Phone, message, log)
	if err != nil {
		return
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
			time.Time(record.DateStart).Format("2006-01-02 15:04:05"),
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
