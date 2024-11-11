## ТГ бот для стоматологии 
тестовый бот - https://t.me/dental_pro_pushkino_test_bot


### Команды бота

- start - Приветствие и начало работы
- record - Запись на прием к стоматологу
- move_record - Перенести запись (пока нет)
- delete_record - Удалить запись на прием
- myrecords - Получить информацию о предстоящих визитах 
- change_name - Изменить имя в системе
- cancel - Отменить последнее действие и вернуться к началу


### Клиентские ресурсы
- [CRM Dental Pro](https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/)

### Unit тесты
В unit тестах текущее время считается как 2024-11-09 17:00:00 по МСК

### Configurations

Создайте файл `.env` в директории `configs/` и добавьте в него следующие параметры:

| Переменная           | Описание                                                | Значение по умолчанию |
|----------------------|---------------------------------------------------------|------------------------|
| `DEBUG`              | Включение режима отладки (`true` / `false`)              | `false`               |
| `TEST`               | Режим тестирования (`true` / `false`)                    | `false`               |
| `TELEGRAM_BOT_TOKEN` | Токен вашего Telegram бота                               |                        |
| `DATABASE_URL`       | URL для подключения к основной базе данных               |                        |
| `TEST_DATABASE_URL`  | URL для подключения к тестовой базе данных               |                        |
| `BRANCH_ID`          | Идентификатор филиала                                    | `3`                    |
| `LOCATION`           | Часовой пояс                                            | `"Europe/Moscow"`     |
| `DENTAL_PRO_TOKEN`   | Токен API для интеграции с DentalPro                     |                        |
| `DENTAL_PRO_SECRET`  | Секретный ключ для DentalPro                             |                        |
