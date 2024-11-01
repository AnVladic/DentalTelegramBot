package bot

import (
	"database/sql"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
	"main/internal/database"
)

type TelegramBotHandler struct {
	bot             *tgbotapi.BotAPI
	userTexts       UserTexts
	dentalProClient crm.IDentalProClient
	db              *sql.DB
}

func NewTelegramBotHandler(
	bot *tgbotapi.BotAPI, userTexts UserTexts, dentalProClient crm.IDentalProClient, db *sql.DB,
) *TelegramBotHandler {
	handler := &TelegramBotHandler{
		bot: bot, userTexts: userTexts, dentalProClient: dentalProClient, db: db,
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
	repository := database.UserRepository{Db: h.db}
	err := repository.CreateUser(&user)
	if err != nil {
		logrus.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
	}

	_, _ = h.Send(response, true)
}

func (h *TelegramBotHandler) NoAuthRegisterCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	ok, err := h.GetPhoneNumber(message, chatState)
	if err != nil {
		_ = fmt.Errorf("GetPhoneNumber error %w", err)
		return
	}
	if !ok {
		chatState.UpdateChatState(h.NoAuthRegisterCommandHandler)
		return
	}
	h.RegisterCommandHandler(message, chatState)
}

func (h *TelegramBotHandler) RegisterCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	logrus.Print("/register command")
	log := logrus.WithFields(logrus.Fields{
		"module": "bot",
		"func":   "RegisterCommandHandler",
	})

	repository := database.UserRepository{Db: h.db}
	_, err := repository.GetUserByTelegramID(message.From.ID)
	if errors.Is(err, sql.ErrNoRows) {
		h.RequestPhoneNumber(message)
		chatState.UpdateChatState(h.NoAuthRegisterCommandHandler)
		return
	} else if err != nil {
		log.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
	}

	go func() {
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Wait)
		newMsg, err := h.Send(response, true)
		if err != nil {
			logrus.Error(err)
			return
		}
		h.ChangeToDoctorsMarkup(newMsg)
	}()
}

func (h *TelegramBotHandler) CancelCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	logrus.Print("/cancel command")
	chatState.UpdateChatState(nil)
	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Cancel)
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

	phoneNumber := message.Contact.PhoneNumber
	repository := database.UserRepository{Db: h.db}
	err := repository.UpsertPhoneByTelegramID(message.From.ID, phoneNumber)
	if err != nil {
		logrus.Error(err)
		response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError)
		_, _ = h.Send(response, false)
		return false, err
	}
	return true, nil
}
