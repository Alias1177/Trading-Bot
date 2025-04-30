package bot

import (
	"Trading-Bot/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	"log"
)

type Qwestion struct {
	First  string
	Second string
}

// UserState хранит состояние диалога с пользователем
type UserState struct {
	Step         int
	FirstAnswer  string
	SecondAnswer string
}

func Start(cfg *config.Config, db *sqlx.DB) {
	qwestion := Qwestion{
		First:  "Ваша почта?",
		Second: "Ваша Валютная пара?",
	}

	bot, err := tgbotapi.NewBotAPI(cfg.ApiTGBotToken)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS usersList (
		    id serial primary key,
    		email varchar not null,
   			currency_pair VARCHAR(255) NOT NULL,
    		price int
		)
	`)

	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Карта для хранения состояний пользователей
	userStates := make(map[int64]*UserState)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		username := update.Message.From.UserName
		text := update.Message.Text

		log.Printf("[%s] %s", username, text)

		// Обработка команды /start
		if text == "/start" {
			// Инициализируем или сбрасываем состояние пользователя
			userStates[userID] = &UserState{Step: 1}

			// Отправляем первый вопрос
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, qwestion.First)
			bot.Send(msg)
			continue
		}

		// Получаем текущее состояние пользователя
		state, exists := userStates[userID]
		if !exists {
			// Если пользователь не начал диалог
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, используйте /start для начала диалога")
			bot.Send(msg)
			continue
		}

		// Обрабатываем ответ в зависимости от текущего шага
		switch state.Step {
		case 1:
			// Сохраняем ответ на первый вопрос
			state.FirstAnswer = text
			state.Step = 2

			// Задаем второй вопрос

			replyMarkup := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("BTC/USD"),
					tgbotapi.NewKeyboardButton("ETH/BTC"),
				),
			)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, qwestion.Second)
			msg.ReplyMarkup = replyMarkup
			bot.Send(msg)

		case 2:
			// Сохраняем ответ на второй вопрос
			state.SecondAnswer = text

			// Сначала скрываем клавиатуру
			removeKeyboard := tgbotapi.NewMessage(update.Message.Chat.ID, "Выбрана пара: "+text)
			removeKeyboard.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true) // параметр true означает, что клавиатура будет скрыта немедленно
			bot.Send(removeKeyboard)

			// Сохраняем данные в БД
			_, err := db.Exec("INSERT INTO usersList (email, currency_pair, price) VALUES ($1, $2, $3)",
				state.FirstAnswer, state.SecondAnswer, 9)

			if err != nil {
				log.Printf("Error inserting into database: %v", err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при сохранении ваших ответов")
				bot.Send(msg)
				delete(userStates, userID) // Сбрасываем состояние в случае ошибки
				continue
			}

			// Отправляем сообщение об успешном сохранении и счет на оплату
			message := tgbotapi.NewMessage(update.Message.Chat.ID, "Ваши данные успешно сохранены. Секундочку, высылаю счёт на оплату")
			bot.Send(message)

			// Сбрасываем состояние пользователя
			delete(userStates, userID)
		}
	}
}
