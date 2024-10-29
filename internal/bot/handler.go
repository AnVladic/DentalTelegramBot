package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"main/internal/crm"
)

type TelegramBotHandler struct {
	bot             *tgbotapi.BotAPI
	userTexts       UserTexts
	dentalProClient crm.IDentalProClient
}

func NewTelegramBotHandler(bot *tgbotapi.BotAPI, userTexts UserTexts, dentalProClient crm.IDentalProClient) *TelegramBotHandler {
	handler := &TelegramBotHandler{bot: bot, userTexts: userTexts, dentalProClient: dentalProClient}
	return handler
}

func (h *TelegramBotHandler) StartCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	logrus.Print("/start command")
	response := tgbotapi.NewMessage(message.Chat.ID, h.userTexts.Welcome)
	_, _ = h.Send(response, true)
}

func (h *TelegramBotHandler) RegisterCommandHandler(message *tgbotapi.Message, chatState *TelegramChatState) {
	logrus.Print("/register command")

	h.RequestPhoneNumber(message)
	chatState.UpdateChatState(h.GetPhoneNumber)

	//doctors, err := h.dentalProClient.DoctorsList()
	//if err != nil {
	//	_, _ = h.Send(tgbotapi.NewMessage(message.Chat.ID, h.userTexts.InternalError), false)
	//	logrus.Print(err)
	//	return
	//}

	//response := tgbotapi.NewMessage(message.Chat.ID, "Пожалуйста, выберите врача для записи. "+
	//	"Вы можете выбрать из доступных специалистов ниже 👇")
	//keyboard := tgbotapi.InlineKeyboardMarkup{}
	//for _, doctor := range doctors {
	//	data := TelegramBotDoctorCallbackData{
	//		CallbackData: CallbackData{"select_doctor"},
	//		DoctorID:     doctor.ID,
	//	}
	//	bytesData, _ := json.Marshal(data)
	//	title := fmt.Sprintf(
	//		"%s - %s", doctor.FIO, strings.Join(GetMapValues(doctor.Departments), ", "))
	//	btn := tgbotapi.NewInlineKeyboardButtonData(title, string(bytesData))
	//	row := []tgbotapi.InlineKeyboardButton{btn}
	//	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	//}
	//response.ReplyMarkup = keyboard
	//_, _ = h.Send(response, true)
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

func (h *TelegramBotHandler) GetPhoneNumber(message *tgbotapi.Message, chatState *TelegramChatState) {
	if message.Contact == nil {
		text := "📲 Пожалуйста, нажмите кнопку <b>📞 Отправить номер телефона</b>, \n\n" +
			"Если передумали, введите команду /cancel ❌"
		response := tgbotapi.NewMessage(message.Chat.ID, text)
		response.ReplyMarkup = h.RequestContactKeyboard()
		response.ParseMode = "HTML"
		_, _ = h.Send(response, true)
		chatState.UpdateChatState(h.GetPhoneNumber)
		return
	}

	if message.Contact != nil {
		phoneNumber := message.Contact.PhoneNumber
		fmt.Println(phoneNumber)
	}
}
