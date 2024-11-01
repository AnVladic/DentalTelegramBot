package bot

type UserTexts struct {
	Welcome             string
	Cancel              string
	Calendar            string
	PhoneNumberRequest  string
	Back                string
	Wait                string
	ChooseDoctor        string
	DontHasAppointments string
	ChooseAppointments  string

	InternalError string
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

		Calendar: "Выберите нужный день",

		PhoneNumberRequest: "Пожалуйста, укажите ваш номер телефона 📱. Он понадобится для подтверждения вашей регистрации и редактирования записи.",

		Back: "Назад",

		ChooseDoctor: "Пожалуйста, выберите врача для записи. Вы можете выбрать из доступных специалистов ниже 👇",

		DontHasAppointments: "К сожалению, у врача %s пока нет доступных приемов 😔.",

		ChooseAppointments: "Пожалуйста, выберите желаемый прием 🌟.",

		Wait: "Секунду...",

		InternalError: "😔 Внутренняя ошибка сервера. Пожалуйста, попробуйте позже. " +
			"Спасибо за понимание! 🙏",
	}
}
