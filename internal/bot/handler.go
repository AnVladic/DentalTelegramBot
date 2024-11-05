package bot

import (
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
	"main/internal/database"
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
	err := repository.CreateUser(&user)
	if err != nil {
		logrus.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
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
