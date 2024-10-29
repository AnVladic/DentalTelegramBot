package bot

type UserTexts struct {
	Welcome            string
	Cancel             string
	Calendar           string
	PhoneNumberRequest string
	InternalError      string
}

func NewUserTexts() *UserTexts {
	return &UserTexts{
		Welcome: `Привет! 👋 Добро пожаловать в нашу стоматологическую клинику.
    
Вот что я могу для вас сделать:
1. /register 📅 Записаться на приём.
2. /myrecord 📝 Получить информацию о предстоящих визитах.
    
Для записи на приём просто отправьте команду /register или выберите нужный пункт в меню.

Если нужна помощь, используйте команду /help.`,

		Cancel: "Мы успешно вернулись в начало",

		Calendar: "Выберите нужный день\n🟢 Свободные дни",

		PhoneNumberRequest: "Чтобы мы могли с вами взаимодействовать дальше, пожалуйста, " +
			"поделитесь номером телефона 📱",

		InternalError: "😔 Внутренняя ошибка сервера. Пожалуйста, попробуйте позже. " +
			"Спасибо за понимание! 🙏",
	}
}
