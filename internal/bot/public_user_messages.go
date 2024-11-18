package bot

type UserTexts struct {
	PositiveAnswers []string
	NegativeAnswers []string

	Welcome                  string
	ExistWelcome             string
	Cancel                   string
	Calendar                 string
	PhoneNumberRequest       string
	Back                     string
	Wait                     string
	ChooseDoctor             string
	DontHasAppointments      string
	ChooseAppointments       string
	DontHasIntervals         string
	ChooseInterval           string
	Approve                  string
	ApproveRegister          string
	ApproveRegisterTimeLimit string
	HasSameRecord            string
	ContactsAddedSuccess     string
	ChangeName               string

	ChangeLastNameRequest  string
	ChangeFirstNameRequest string
	ChangeNameSucceed      string

	RegisterIntervalError string
	RegisterSuccess       string

	HasNoRecords   string
	RecordList     string
	RecordItem     string
	RecordWillItem string

	DeleteRecords              string
	DeleteRecordItem           string
	ApproveDeleteRecord        string
	HasNoDeleteRecord          string
	CancelDeleteRecord         string
	UnknownApproveDeleteRecord string
	SuccessDeleteRecord        string

	InternalError string
}

func NewUserTexts() *UserTexts {
	return &UserTexts{
		PositiveAnswers: []string{
			"✅ Подтвердить",
			"Подтвердить",
			"Да",
			"Yes",
			"Y",
			"Ok",
			"Ок",
		},

		NegativeAnswers: []string{
			"Отменить",
			"Нет",
			"Не",
			"No",
			"N",
		},

		Welcome: `Привет! 👋 Добро пожаловать в нашу стоматологическую клинику в "Олимп" Софрино 🦷✨ 

Вот что я могу для вас сделать:
- 🗓️ /record — Запись на приём к стоматологу
- 🗑️ /delete_record — Удалить запись на приём
- 📋 /myrecords — Получить информацию о предстоящих визитах
- ✏️ /change_name — Изменить имя в системе
- ❌ /cancel — Отменить последнее действие и вернуться к началу

Для записи на приём просто отправьте команду /record или выберите нужный пункт в меню.`,

		ExistWelcome: `Во время использования бота мы можем запросить ваш номер телефона ☎️.
Если он совпадает с номером, который вы указывали при первом визите, все данные будут синхронизированы автоматически.`,

		Cancel: "Мы успешно вернулись в начало",

		Calendar: "Выберите нужный день",

		PhoneNumberRequest: "Пожалуйста, укажите ваш номер телефона 📱. Он понадобится для подтверждения вашей регистрации и редактирования записи.\n\n" +
			"Нажмите кнопку <b>📞 Отправить номер телефона</b>",

		Back: "Назад",

		ChooseDoctor: "Пожалуйста, выберите врача для записи. Вы можете выбрать из доступных специалистов ниже 👇",

		DontHasAppointments: "К сожалению, у врача %s пока нет доступных приемов 😔.",

		ChooseAppointments: "Пожалуйста, выберите желаемый прием 🌟.",

		DontHasIntervals: "День %s\nВрач %s\n%s\n\nК сожалению, у врача %s пока нет свободных интервалов в этот день. 😔🗓️",

		ChooseInterval: "День %s\nВрач %s\n%s\n\nПожалуйста, выберите свободное время. 🕒✨",

		Approve: "✅ Подтвердить",

		ApproveRegisterTimeLimit: "⚠️ Упс! Вы не можете записаться на уже прошедшую дату и время",

		ApproveRegister: "Стоматологическая клиника \"Олимп\" в Софрино\n\n" +
			"📅 Дата и время: <b><i>%s</i></b>\n👨‍⚕️ Врач: <b><i>%s</i></b>" +
			"\n🦷 На прием: <b><i>%s (%d мин)</i></b>\n\nВы будете записаны как: <b><i>%s %s</i></b>" +
			"\n\nПожалуйста, подтвердите, что все верно.",

		HasSameRecord: "К сожалению, вы не можете записаться к этому врачу, так как уже состоите в списке записавшихся 🩺❗ к нему",

		ContactsAddedSuccess: "📞 Ваш номер телефона успешно добавлен!\nВы можете продолжить регистрацию.",

		ChangeName: "Изменить имя",

		ChangeFirstNameRequest: "🗝 Пожалуйста, укажите ваше имя.",

		ChangeLastNameRequest: "🗝 Пожалуйста, теперь укажите фамилию.",

		ChangeNameSucceed: "🎉 Ваше имя успешно изменено на <b><i>%s %s</i></b>!",

		RegisterIntervalError: "К сожалению, выбранный интервал недоступен для записи 😔. Пожалуйста, выберите другой 🗓️.",

		RegisterSuccess: "Вы успешно записались на прием! 🎉\n\n" +
			"Стоматологическая клиника \"Олимп\" в Софрино\n\n" +
			"📅 Дата и время: <b><i>%s %s</i></b>\n👨‍⚕️ Врач: <b><i>%s</i></b>" +
			"\n🦷 На прием: <b><i>%s (%d мин)</i></b>\n\nВы записаны как: <b><i>%s %s</i></b>\n\n" +
			"Воспользуйтесь командой:\n" +
			"\t/delete_record ❌ — если хотите удалить запись\n\n" +
			"Ждем вас! 😊",

		Wait: "Секунду...",

		InternalError: "😔 Внутренняя ошибка сервера. Пожалуйста, попробуйте позже. " +
			"Спасибо за понимание! 🙏",

		HasNoRecords: "Похоже, у вас нет записей 📅",
		RecordList: "Список ваших записей в стоматологическую клинику \"Олимп\" в Софрино\n" +
			"🔴 - Предстоящие записи\n\n",
		RecordItem: "Запись №%d\n📅 Дата и время: <b><i>%s</i></b>\n👨‍⚕️ Врач: <b><i>%s - %s</i></b>" +
			"\n🦷 На прием: <b><i>%s (%d мин)</i></b>",
		RecordWillItem: "🔴 Запись №%d\n📅 Дата и время: <b><i>%s</i></b>\n👨‍⚕️ Врач: <b><i>%s - %s</i></b>" +
			"\n🦷 На прием: <b><i>%s (%d мин)</i></b>",

		DeleteRecords:       "Выберите запись, которую хотите удалить ❌",
		DeleteRecordItem:    "Запись №%d: %s %s",
		ApproveDeleteRecord: "Вы хотите удалить запись — %s, %s 🗓️.\n\nПодтвердить удаление? ✅",
		HasNoDeleteRecord:   "К сожалению, такой записи не найдено 😕",
		CancelDeleteRecord:  "Удаление записи — %s, %s, отменено ❌",
		UnknownApproveDeleteRecord: `Пожалуйста, напишите 'Да' или 'Нет', либо соответсвующую нажмите на кнопку 😊

Если хотите вернуться и отменить действие, напишите /cancel`,
		SuccessDeleteRecord: "Запись — %s, %s, успешно удалена ✅",
	}
}
