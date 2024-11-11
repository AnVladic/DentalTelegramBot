package bot

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"log"
	"main/internal/crm"
	"os"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/mock"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type MockTelegramAPI struct {
	chatID    int64
	messageID int
	mock.Mock
	Updates chan tgbotapi.Update
}

type TestCase struct {
	userMessage func() tgbotapi.Update
	expected    func() []tgbotapi.Chattable
}

type TestNow struct{}

const BranchId int64 = 3
const UserId int64 = 12345

var LOCATION, _ = time.LoadLocation("Europe/Moscow")

func (m *MockTelegramAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	args := m.Called(c)
	return args.Get(0).(tgbotapi.Message), args.Error(1)
}

func (m *MockTelegramAPI) GetUpdatesChan(_ tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return m.Updates
}

func (t *TestNow) Now() time.Time {
	return time.Date(2024, 11, 9, 17, 0, 0, 0, LOCATION)
}

func createBot() (*Router, *MockTelegramAPI, *sql.DB) {
	err := godotenv.Load("../../configs/.env")
	if err != nil {
		panic(fmt.Errorf("error loading .env file: %w", err))
	}

	testDB, err := sql.Open("pgx", os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		log.Fatalf("Не удалось подключиться к тестовой базе данных: %v", err)
	}
	err = testDB.Ping()

	if err != nil {
		logrus.Panic(err)
	}
	testTGBot := &MockTelegramAPI{Updates: make(chan tgbotapi.Update, 1)}

	userTexts := NewUserTexts()
	dentalProClientTest := crm.NewDentalProClient("", "", true, "../crm")
	telegramBotHandler := NewTelegramBotHandler(
		testTGBot, *userTexts, dentalProClientTest, testDB, BranchId, LOCATION, &TestNow{},
	)
	return NewRouter(testTGBot, telegramBotHandler, true), testTGBot, testDB
}

func createTestMessage(chatID int64, messageID int, text string) *tgbotapi.Message {
	return &tgbotapi.Message{
		MessageID: messageID,
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		Text: text,
		From: &tgbotapi.User{
			ID:       UserId,
			UserName: "testuser",
		},
	}
}

func createTestQuery(chatID int64, messageID int, data string) *tgbotapi.CallbackQuery {
	return &tgbotapi.CallbackQuery{
		From: &tgbotapi.User{
			ID:       UserId,
			UserName: "testuser",
		},
		Message: createTestMessage(chatID, messageID, ""),
		Data:    data,
	}
}

func createChat(chatID int64) *tgbotapi.Chat {
	return &tgbotapi.Chat{
		ID: chatID,
	}
}

func clearAllTables(db *sql.DB) error {
	_, err := db.Exec(`
        DO $$ DECLARE
            r RECORD;
        BEGIN
            FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
                EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
            END LOOP;
        END $$;
    `)
	if err != nil {
		log.Printf("Ошибка при очистке таблиц: %v", err)
	}
	return err
}

func createContact() *tgbotapi.Contact {
	return &tgbotapi.Contact{
		PhoneNumber: "79999999999",
		LastName:    "Ivanov",
		FirstName:   "Ivan",
		UserID:      UserId,
	}
}

func checkCases(t *testing.T, router *Router, mockBot *MockTelegramAPI, chatID int64, cases []TestCase) {
	for i, _case := range cases {
		expectedMessages := _case.expected()
		mockBot.ExpectedCalls = nil
		for _, expectedMessage := range expectedMessages {
			var args interface{} = expectedMessage
			if expectedMessage == nil {
				args = mock.Anything
			}
			mockBot.On("Send", args).Return(
				*createTestMessage(chatID, mockBot.messageID, ""), nil,
			).Once()
		}
		router.TestWG.Add(1)
		mockBot.Updates <- _case.userMessage()
		router.TestWG.Wait()
		for _, expectedMessage := range expectedMessages {
			var expected interface{} = expectedMessage
			if expectedMessage == nil {
				expected = mock.Anything
			}
			if !mockBot.AssertCalled(t, "Send", expected) {
				return
			}
		}
		fmt.Printf("pass case #%d\n", i+1)
	}
}

func TestRegisterHandle(t *testing.T) {
	var chatID int64 = 1
	var router, mockBot, db = createBot()
	defer func(testDB *sql.DB) {
		_ = testDB.Close()
	}(db)

	err := clearAllTables(db)
	if err != nil {
		fmt.Println(err)
	}

	stopChan := make(chan struct{})
	go router.StartListening(stopChan)

	testCases := []TestCase{
		{ // 1
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 2, "/myrecords")
				message.Entities = []tgbotapi.MessageEntity{
					{Type: "bot_command", Length: len([]rune(message.Text))},
				}
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				expectedMessage := tgbotapi.NewMessage(chatID, "Пожалуйста, укажите ваш номер телефона 📱. Он понадобится для подтверждения вашей регистрации и редактирования записи.\n\nНажмите кнопку <b>📞 Отправить номер телефона</b>")
				phoneButton := tgbotapi.KeyboardButton{
					Text:           "📞 Отправить номер телефона",
					RequestContact: true,
				}
				keyboard := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(phoneButton),
				)
				keyboard.OneTimeKeyboard = true
				expectedMessage.ReplyMarkup = keyboard
				expectedMessage.ParseMode = HTML
				return []tgbotapi.Chattable{expectedMessage}
			},
		},
		{ // 2
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 3, "random text")
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				expectedMessage := tgbotapi.NewMessage(chatID, "📲 Пожалуйста, нажмите кнопку <b>📞 Отправить номер телефона</b>, \n\nЕсли передумали, введите команду /cancel ❌")
				phoneButton := tgbotapi.KeyboardButton{
					Text:           "📞 Отправить номер телефона",
					RequestContact: true,
				}
				keyboard := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(phoneButton),
				)
				keyboard.OneTimeKeyboard = true
				expectedMessage.ReplyMarkup = keyboard
				expectedMessage.ParseMode = HTML
				return []tgbotapi.Chattable{expectedMessage}
			},
		},
		{ // 3
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 4, "")
				message.Contact = createContact()
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				return []tgbotapi.Chattable{tgbotapi.NewMessage(chatID, "Похоже, у вас нет записей 📅")}
			},
		},
		{ // 4
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 5, "/start")
				message.Entities = []tgbotapi.MessageEntity{
					{Type: "bot_command", Length: len([]rune(message.Text))},
				}
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				return []tgbotapi.Chattable{tgbotapi.NewMessage(chatID, `Привет! 👋 Добро пожаловать в нашу стоматологическую клинику 🦷✨ 

Вот что я могу для вас сделать:
- 🗓️ /record — Запись на приём к стоматологу
- 🔄 /move_record — Перенести запись
- 🗑️ /delete_record — Удалить запись на приём
- 📋 /myrecords — Получить информацию о предстоящих визитах
- ✏️ /change_name — Изменить имя в системе
- ❌ /cancel — Отменить последнее действие и вернуться к началу

Для записи на приём просто отправьте команду /register или выберите нужный пункт в меню.`)}
			},
		},
		{ // 5
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 6, "/record")
				message.Chat = createChat(chatID)
				message.Entities = []tgbotapi.MessageEntity{
					{Type: "bot_command", Length: len([]rune(message.Text))},
				}
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				text := "Пожалуйста, выберите врача для записи. Вы можете выбрать из доступных специалистов ниже 👇"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("Подаева С.Е. - Терапевты", `{"command":"select_doctor","d":2}`)},
					{tgbotapi.NewInlineKeyboardButtonData("Новикова Н.В. - Гигиенисты", `{"command":"select_doctor","d":12}`)},
					{tgbotapi.NewInlineKeyboardButtonData("Коченова Е.Д. - Гигиенисты", `{"command":"select_doctor","d":14}`)},
					{tgbotapi.NewInlineKeyboardButtonData("Галустян А.В. - Хирурги, Терапевты", `{"command":"select_doctor","d":15}`)},
					{tgbotapi.NewInlineKeyboardButtonData("Нифанов А.А. - Хирурги, Ортопеды", `{"command":"select_doctor","d":16}`)},
					{tgbotapi.NewInlineKeyboardButtonData("Егиазарян А.А. - Терапевты, Ортодонты, Детская терапия", `{"command":"select_doctor","d":18}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 0, text, keyboard)
				return []tgbotapi.Chattable{
					tgbotapi.NewMessage(chatID, "Секунду..."),
					exceptedMsg,
				}
			},
		},
		{ // 6
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 0, `{"command":"select_doctor","d":14}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "К сожалению, у врача Коченова Е.Д. пока нет доступных приемов 😔."
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"doctors"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 0, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 7
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 0, `{"command":"select_doctor","d":2}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "Пожалуйста, выберите желаемый прием 🌟."
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("(15 мин.) Проведение профосмотра терапевта.", `{"command":"appointment","a":41}`)},
					{tgbotapi.NewInlineKeyboardButtonData("(30 мин.) Повторная консультация терапевта.", `{"command":"appointment","a":86}`)},
					{tgbotapi.NewInlineKeyboardButtonData("(60 мин.) Повторная консультация + лечение терапевта.", `{"command":"appointment","a":25}`)},
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"doctors"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 0, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 8
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 0, `{"command":"appointment","a":25}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "Выберите нужный день - Подаева С.Е.\nПовторная консультация + лечение терапевта.\n🟢 Доступные дни"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("November 2024", "1")},
					{
						tgbotapi.NewInlineKeyboardButtonData("Пн", `Пн`),
						tgbotapi.NewInlineKeyboardButtonData("Вт", `Вт`),
						tgbotapi.NewInlineKeyboardButtonData("Ср", `Ср`),
						tgbotapi.NewInlineKeyboardButtonData("Чт", `Чт`),
						tgbotapi.NewInlineKeyboardButtonData("Пт", `Пт`),
						tgbotapi.NewInlineKeyboardButtonData("Сб", `Сб`),
						tgbotapi.NewInlineKeyboardButtonData("Вс", `Вс`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData(" ", `1`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `2`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `3`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `4`),
						tgbotapi.NewInlineKeyboardButtonData("1", `{"command":"day","dt":"2024.11.1","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("2", `{"command":"day","dt":"2024.11.2","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("3", `{"command":"day","dt":"2024.11.3","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("4", `{"command":"day","dt":"2024.11.4","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("5", `{"command":"day","dt":"2024.11.5","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("6", `{"command":"day","dt":"2024.11.6","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("7", `{"command":"day","dt":"2024.11.7","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("8", `{"command":"day","dt":"2024.11.8","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 9", `{"command":"day","dt":"2024.11.9","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("10", `{"command":"day","dt":"2024.11.10","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("11", `{"command":"day","dt":"2024.11.11","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 12", `{"command":"day","dt":"2024.11.12","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("13", `{"command":"day","dt":"2024.11.13","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("14", `{"command":"day","dt":"2024.11.14","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 15", `{"command":"day","dt":"2024.11.15","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 16", `{"command":"day","dt":"2024.11.16","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("17", `{"command":"day","dt":"2024.11.17","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("🟢 18", `{"command":"day","dt":"2024.11.18","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 19", `{"command":"day","dt":"2024.11.19","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("20", `{"command":"day","dt":"2024.11.20","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("21", `{"command":"day","dt":"2024.11.21","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 22", `{"command":"day","dt":"2024.11.22","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("23", `{"command":"day","dt":"2024.11.23","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("24", `{"command":"day","dt":"2024.11.24","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("25", `{"command":"day","dt":"2024.11.25","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 26", `{"command":"day","dt":"2024.11.26","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("27", `{"command":"day","dt":"2024.11.27","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("28", `{"command":"day","dt":"2024.11.28","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 29", `{"command":"day","dt":"2024.11.29","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 30", `{"command":"day","dt":"2024.11.30","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `1`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData(">", `{"command":"switch_timesheet_month","m":"2024.12","d":2}`),
					},
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"appointments"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 0, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 9
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 0, `{"command":"day","dt":"2024.11.9","s":0}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "День 09.11.2024\nВрач Подаева С.Е.\nПовторная консультация + лечение терапевта.\n\nПожалуйста, выберите свободное время. 🕒✨"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{
						tgbotapi.NewInlineKeyboardButtonData("❌ 16:00 - 17:00", `{"command":"interval","s":"16:00"}`),
						tgbotapi.NewInlineKeyboardButtonData("18:00 - 19:00", `{"command":"interval","s":"18:00"}`),
					},
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"calendar"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 0, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 9
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 0, `{"command":"day","dt":"2024.11.9","s":0}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "День 09.11.2024\nВрач Подаева С.Е.\nПовторная консультация + лечение терапевта.\n\nПожалуйста, выберите свободное время. 🕒✨"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{
						tgbotapi.NewInlineKeyboardButtonData("❌ 16:00 - 17:00", `{"command":"interval","s":"16:00"}`),
						tgbotapi.NewInlineKeyboardButtonData("18:00 - 19:00", `{"command":"interval","s":"18:00"}`),
					},
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"calendar"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 0, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 10
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 0, `{"command":"interval","s":"16:00"}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "⚠️ Упс! Вы не можете записаться на уже прошедшую дату и время"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"calendar"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 0, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 12
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 0, `{"command":"interval","s":"18:00"}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := `Стоматологическая клиника "Олимп" в Софрино

📅 Дата и время: <b><i>2024-11-09 18:00</i></b>
👨‍⚕️ Врач: <b><i>Подаева С.Е.</i></b>
🦷 На прием: <b><i>Повторная консультация + лечение терапевта. (60 мин)</i></b>

Вы будете записаны как: <b><i>Ivanov Ivan</i></b>

Пожалуйста, подтвердите, что все верно.`
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("Изменить имя", `{"command":"change_name","d":"register"}`)},
					{
						tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", `{"command":"approve","d":"register"}`),
						tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"calendar"}`),
					},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 0, text, keyboard)
				exceptedMsg.ParseMode = HTML
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 13
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 0, `{"command":"approve","d":"register"}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := `Вы успешно записались на прием! 🎉

Стоматологическая клиника "Олимп" в Софрино

📅 Дата и время: <b><i>2024-11-09 18:00:00</i></b>
👨‍⚕️ Врач: <b><i>Подаева С.Е.</i></b>
🦷 На прием: <b><i>Повторная консультация + лечение терапевта. (60 мин)</i></b>

Вы записаны как: <b><i>Ivanov Ivan</i></b>

Воспользуйтесь командами:
	/move_record 🔄 — если хотите перенести запись
	/delete_record ❌ — если хотите удалить запись

Ждем вас! 😊`
				exceptedMsg := tgbotapi.NewEditMessageText(chatID, 0, text)
				exceptedMsg.ParseMode = HTML
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},

		// Важное условие, что один клиент может записаться только к одному врачу
		{ // 14
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 10, `{"command":"select_doctor","d":2}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "К сожалению, вы не можете записаться к этому врачу, так как уже состоите в списке записавшихся 🩺❗ к нему"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"doctors"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 10, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},

		{ // 15
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"select_doctor","d":12}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "Пожалуйста, выберите желаемый прием 🌟."
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("(30 мин.) Повторная консультация терапевта.", `{"command":"appointment","a":86}`)},
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"doctors"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 15, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 16
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"appointment","a":86}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				return []tgbotapi.Chattable{nil}
			},
		},
		{ // 17
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"switch_timesheet_month","m":"2024.12","d":12}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "Выберите нужный день - Новикова Н.В.\nПовторная консультация терапевта.\n🟢 Доступные дни"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("December 2024", "1")},
					{
						tgbotapi.NewInlineKeyboardButtonData("Пн", `Пн`),
						tgbotapi.NewInlineKeyboardButtonData("Вт", `Вт`),
						tgbotapi.NewInlineKeyboardButtonData("Ср", `Ср`),
						tgbotapi.NewInlineKeyboardButtonData("Чт", `Чт`),
						tgbotapi.NewInlineKeyboardButtonData("Пт", `Пт`),
						tgbotapi.NewInlineKeyboardButtonData("Сб", `Сб`),
						tgbotapi.NewInlineKeyboardButtonData("Вс", `Вс`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData(" ", `1`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `2`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `3`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `4`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `5`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `6`),
						tgbotapi.NewInlineKeyboardButtonData("1", `{"command":"day","dt":"2024.12.1","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("2", `{"command":"day","dt":"2024.12.2","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("3", `{"command":"day","dt":"2024.12.3","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("4", `{"command":"day","dt":"2024.12.4","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("5", `{"command":"day","dt":"2024.12.5","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("6", `{"command":"day","dt":"2024.12.6","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("7", `{"command":"day","dt":"2024.12.7","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("8", `{"command":"day","dt":"2024.12.8","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("9", `{"command":"day","dt":"2024.12.9","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("10", `{"command":"day","dt":"2024.12.10","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("🟢 11", `{"command":"day","dt":"2024.12.11","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("12", `{"command":"day","dt":"2024.12.12","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("13", `{"command":"day","dt":"2024.12.13","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("14", `{"command":"day","dt":"2024.12.14","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("15", `{"command":"day","dt":"2024.12.15","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("16", `{"command":"day","dt":"2024.12.16","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("17", `{"command":"day","dt":"2024.12.17","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("18", `{"command":"day","dt":"2024.12.18","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("19", `{"command":"day","dt":"2024.12.19","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("20", `{"command":"day","dt":"2024.12.20","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("21", `{"command":"day","dt":"2024.12.21","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("22", `{"command":"day","dt":"2024.12.22","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("23", `{"command":"day","dt":"2024.12.23","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("24", `{"command":"day","dt":"2024.12.24","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("25", `{"command":"day","dt":"2024.12.25","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("26", `{"command":"day","dt":"2024.12.26","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("27", `{"command":"day","dt":"2024.12.27","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("28", `{"command":"day","dt":"2024.12.28","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("29", `{"command":"day","dt":"2024.12.29","s":0}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("30", `{"command":"day","dt":"2024.12.30","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData("31", `{"command":"day","dt":"2024.12.31","s":0}`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `1`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `2`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `3`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `4`),
						tgbotapi.NewInlineKeyboardButtonData(" ", `5`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("<", `{"command":"switch_timesheet_month","m":"2024.11","d":12}`),
						tgbotapi.NewInlineKeyboardButtonData(">", `{"command":"switch_timesheet_month","m":"2025.1","d":12}`),
					},
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"appointments"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 15, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 18
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"day","dt":"2024.12.8","s":0}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "День 08.12.2024\nВрач Новикова Н.В.\nПовторная консультация терапевта.\n\n" +
					"К сожалению, у врача Новикова Н.В. пока нет свободных интервалов в этот день. 😔🗓️"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"calendar"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 15, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 19
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"day","dt":"2024.12.11","s":0}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := "День 11.12.2024\nВрач Новикова Н.В.\nПовторная консультация терапевта.\n\nПожалуйста, выберите свободное время. 🕒✨"
				keyboard := tgbotapi.InlineKeyboardMarkup{}
				keyboard.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{
					{
						tgbotapi.NewInlineKeyboardButtonData("11:10 - 11:40", `{"command":"interval","s":"11:10"}`),
						tgbotapi.NewInlineKeyboardButtonData("11:40 - 12:10", `{"command":"interval","s":"11:40"}`),
						tgbotapi.NewInlineKeyboardButtonData("12:10 - 12:40", `{"command":"interval","s":"12:10"}`),
					},
					{
						tgbotapi.NewInlineKeyboardButtonData("12:40 - 13:10", `{"command":"interval","s":"12:40"}`),
						tgbotapi.NewInlineKeyboardButtonData("13:10 - 13:40", `{"command":"interval","s":"13:10"}`),
						tgbotapi.NewInlineKeyboardButtonData("13:40 - 14:10", `{"command":"interval","s":"13:40"}`),
					},
					{tgbotapi.NewInlineKeyboardButtonData("Назад", `{"command":"back","b":"calendar"}`)},
				}
				exceptedMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, 15, text, keyboard)
				return []tgbotapi.Chattable{exceptedMsg}
			},
		},
		{ // 21
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"interval","s":"12:40"}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				return []tgbotapi.Chattable{nil}
			},
		},
		{ // 21
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"approve","d":"register"}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				return []tgbotapi.Chattable{nil}
			},
		},
		{ // 22
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 22, "/myrecords")
				message.Entities = []tgbotapi.MessageEntity{
					{Type: "bot_command", Length: len([]rune(message.Text))},
				}
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				text := `Список ваших записей в стоматологическую клинику "Олимп" в Софрино

Запись №1
📅 Дата и время: <b><i>2024-11-09 18:00</i></b>
👨‍⚕️ Врач: <b><i>Подаева С.Е. - Терапевты</i></b>
🦷 На прием: <b><i>Тестовая запись. (0 мин)</i></b>

Запись №2
📅 Дата и время: <b><i>2024-12-11 12:40</i></b>
👨‍⚕️ Врач: <b><i>Новикова Н.В. - Гигиенисты</i></b>
🦷 На прием: <b><i>Тестовая запись. (0 мин)</i></b>`

				msg := tgbotapi.NewMessage(chatID, text)
				msg.ParseMode = HTML
				return []tgbotapi.Chattable{msg}
			},
		},
		{ // 23
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 23, "/delete_record")
				message.Entities = []tgbotapi.MessageEntity{
					{Type: "bot_command", Length: len([]rune(message.Text))},
				}
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				text := "Выберите запись, которую хотите удалить ❌"

				msg := tgbotapi.NewMessage(chatID, text)
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					[]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(
						"Запись №1: 2024-11-09 18:00 Подаева С.Е.",
						`{"command":"del_r","r":1}`),
					},
					[]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(
						"Запись №2: 2024-12-11 12:40 Новикова Н.В.",
						`{"command":"del_r","r":2}`),
					},
				)
				return []tgbotapi.Chattable{msg}
			},
		},
		{ // 24
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"del_r","r":1}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := `Вы хотите удалить запись — 2024-11-09 18:00, Подаева С.Е. 🗓️.

Подтвердить удаление? ✅`

				msg := tgbotapi.NewMessage(chatID, text)
				keyboard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{
					tgbotapi.NewKeyboardButton("✅ Подтвердить"),
					tgbotapi.NewKeyboardButton("Отменить"),
				})
				keyboard.OneTimeKeyboard = true
				msg.ReplyMarkup = keyboard
				return []tgbotapi.Chattable{msg}
			},
		},
		{ // 25
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 25, "нет")
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				text := `Удаление записи — 2024-11-09 18:00, Подаева С.Е., отменено ❌`

				msg := tgbotapi.NewMessage(chatID, text)
				return []tgbotapi.Chattable{msg}
			},
		},
		{ // 26
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"del_r","r":2}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := `Вы хотите удалить запись — 2024-12-11 12:40, Новикова Н.В. 🗓️.

Подтвердить удаление? ✅`

				msg := tgbotapi.NewMessage(chatID, text)
				keyboard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{
					tgbotapi.NewKeyboardButton("✅ Подтвердить"),
					tgbotapi.NewKeyboardButton("Отменить"),
				})
				keyboard.OneTimeKeyboard = true
				msg.ReplyMarkup = keyboard
				return []tgbotapi.Chattable{msg}
			},
		},
		{ // 27
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 25, "да")
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				text := `Запись — 2024-12-11 12:40, Новикова Н.В., успешно удалена ✅`

				msg := tgbotapi.NewMessage(chatID, text)
				return []tgbotapi.Chattable{msg}
			},
		},
		{ // 28
			userMessage: func() tgbotapi.Update {
				callbackQuery := createTestQuery(chatID, 15, `{"command":"del_r","r":123325346}`)
				return tgbotapi.Update{CallbackQuery: callbackQuery}
			},
			expected: func() []tgbotapi.Chattable {
				text := `К сожалению, такой записи не найдено 😕`
				msg := tgbotapi.NewMessage(chatID, text)
				return []tgbotapi.Chattable{msg}
			},
		},
	}
	checkCases(t, router, mockBot, chatID, testCases)
	_ = clearAllTables(db)
	stopChan <- struct{}{}
}

func TestChangeName(t *testing.T) {
	var chatID int64 = 1
	var router, mockBot, db = createBot()
	defer func(testDB *sql.DB) {
		_ = testDB.Close()
	}(db)

	err := clearAllTables(db)
	if err != nil {
		fmt.Println(err)
	}

	stopChan := make(chan struct{})
	go router.StartListening(stopChan)

	testCases := []TestCase{
		{ // 1
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 2, "/change_name")
				message.Entities = []tgbotapi.MessageEntity{
					{Type: "bot_command", Length: len([]rune(message.Text))},
				}
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				expectedMessage := tgbotapi.NewMessage(chatID, "Пожалуйста, укажите ваш номер телефона 📱. Он понадобится для подтверждения вашей регистрации и редактирования записи.\n\nНажмите кнопку <b>📞 Отправить номер телефона</b>")
				phoneButton := tgbotapi.KeyboardButton{
					Text:           "📞 Отправить номер телефона",
					RequestContact: true,
				}
				keyboard := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(phoneButton),
				)
				keyboard.OneTimeKeyboard = true
				expectedMessage.ReplyMarkup = keyboard
				expectedMessage.ParseMode = HTML
				return []tgbotapi.Chattable{expectedMessage}
			},
		},
		{ // 2
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 4, "")
				message.Contact = createContact()
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				return []tgbotapi.Chattable{tgbotapi.NewMessage(chatID, "🗝 Пожалуйста, укажите ваше имя.")}
			},
		},
		{ // 3
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 5, "/cancel")
				message.Entities = []tgbotapi.MessageEntity{
					{Type: "bot_command", Length: len([]rune(message.Text))},
				}
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				message := tgbotapi.NewMessage(chatID, "Мы успешно вернулись в начало")
				message.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				return []tgbotapi.Chattable{message}
			},
		},
		{ // 4
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 6, "/change_name")
				message.Entities = []tgbotapi.MessageEntity{
					{Type: "bot_command", Length: len([]rune(message.Text))},
				}
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				return []tgbotapi.Chattable{tgbotapi.NewMessage(chatID, "🗝 Пожалуйста, укажите ваше имя.")}
			},
		},
		{ // 5
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 7, "Donald")
				message.Contact = createContact()
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				return []tgbotapi.Chattable{tgbotapi.NewMessage(chatID, "🗝 Пожалуйста, теперь укажите фамилию.")}
			},
		},
		{ // 6
			userMessage: func() tgbotapi.Update {
				message := createTestMessage(chatID, 8, "Trump")
				message.Contact = createContact()
				return tgbotapi.Update{Message: message}
			},
			expected: func() []tgbotapi.Chattable {
				message := tgbotapi.NewMessage(chatID, "🎉 Ваше имя успешно изменено на <b><i>Trump Donald</i></b>!")
				message.ParseMode = HTML
				return []tgbotapi.Chattable{message}
			},
		},
	}
	checkCases(t, router, mockBot, chatID, testCases)
	stopChan <- struct{}{}
}
