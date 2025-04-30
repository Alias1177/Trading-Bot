package bot

import (
	"Trading-Bot/config"
	"Trading-Bot/internal/models"
	"Trading-Bot/internal/payment"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	"log"
	"regexp"
)

type Question struct {
	First  string
	Second string
}

// UserState stores the dialog state with the user
type UserState struct {
	Step         int
	Email        string
	CurrencyPair string
	UserID       int64
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func Start(cfg *config.Config, db *sqlx.DB) {
	questions := Question{
		First:  "Ваша почта?",
		Second: "Ваша Валютная пара?",
	}

	// Create a notification channel for payment updates
	paymentChan := make(chan models.PaymentNotification, 100)

	// Initialize Stripe service
	stripeService := payment.NewStripeService(cfg, db, paymentChan)

	// Initialize database tables
	if err := stripeService.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database tables: %v", err)
	}

	// Create users table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users_list (
		    id SERIAL PRIMARY KEY,
    		email VARCHAR NOT NULL,
   			currency_pair VARCHAR(255) NOT NULL,
    		price INT NOT NULL DEFAULT 9,
			chat_id BIGINT NOT NULL,
			paid BOOLEAN DEFAULT FALSE
		)
	`)

	if err != nil {
		log.Fatalf("Error creating users table: %v", err)
	}

	// Start the webhook server for Stripe
	stripeService.StartWebhookServer("8080") // You can change the port as needed

	// Start a goroutine to handle payment notifications
	go handlePaymentNotifications(db, paymentChan)

	// Initialize Telegram bot
	bot, err := tgbotapi.NewBotAPI(cfg.ApiTGBotToken)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Store user states
	userStates := make(map[int64]*UserState)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		chatID := update.Message.Chat.ID
		username := update.Message.From.UserName
		text := update.Message.Text

		log.Printf("[%s] %s", username, text)

		// Handle /start command
		if text == "/start" {
			// Initialize or reset user state
			userStates[userID] = &UserState{Step: 1, UserID: userID}

			// Send first question
			msg := tgbotapi.NewMessage(chatID, questions.First)
			bot.Send(msg)
			continue
		}

		// Get current user state
		state, exists := userStates[userID]
		if !exists {
			// If user hasn't started the dialog
			msg := tgbotapi.NewMessage(chatID, "Пожалуйста, используйте /start для начала диалога")
			bot.Send(msg)
			continue
		}

		// Process response based on current step
		switch state.Step {
		case 1: // Email input
			// Validate email
			if !isValidEmail(text) {
				msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите корректный email в формате example@domain.com")
				bot.Send(msg)
				continue
			}

			// Save email and move to next step
			state.Email = text
			state.Step = 2

			// Ask second question with keyboard
			replyMarkup := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("BTC/USD"),
					tgbotapi.NewKeyboardButton("ETH/BTC"),
				),
			)
			msg := tgbotapi.NewMessage(chatID, questions.Second)
			msg.ReplyMarkup = replyMarkup
			bot.Send(msg)

		case 2: // Currency pair selection
			// Save currency pair
			state.CurrencyPair = text

			// Hide keyboard
			removeKeyboard := tgbotapi.NewMessage(chatID, "Выбрана пара: "+text)
			removeKeyboard.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			bot.Send(removeKeyboard)

			// Save user data to database
			var userDBID int64
			err := db.QueryRow(
				"INSERT INTO users_list (email, currency_pair, price, chat_id, paid) VALUES ($1, $2, $3, $4, $5) RETURNING id",
				state.Email, state.CurrencyPair, 9, chatID, false,
			).Scan(&userDBID)

			if err != nil {
				log.Printf("Error inserting into database: %v", err)
				msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при сохранении ваших ответов")
				bot.Send(msg)
				delete(userStates, userID)
				continue
			}

			// Send payment processing message
			msg := tgbotapi.NewMessage(chatID, "Ваши данные успешно сохранены. Секундочку, высылаю счёт на оплату...")
			bot.Send(msg)

			// Create Stripe checkout session
			checkoutURL, err := stripeService.CreateCheckoutSession(state.Email, state.CurrencyPair, chatID, userDBID)
			if err != nil {
				log.Printf("Error creating checkout session: %v", err)
				errorMsg := tgbotapi.NewMessage(chatID, "Произошла ошибка при создании счета на оплату. Пожалуйста, попробуйте позже.")
				bot.Send(errorMsg)
				delete(userStates, userID)
				continue
			}

			// Send payment link
			paymentMsg := tgbotapi.NewMessage(chatID, "💳 Пожалуйста, оплатите по ссылке ниже:")
			paymentMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("🔒 Оплатить заказ", checkoutURL),
				),
			)
			bot.Send(paymentMsg)

			// Reset user state
			delete(userStates, userID)
		}
	}
}

// handlePaymentNotifications processes completed payments
func handlePaymentNotifications(db *sqlx.DB, notifications chan models.PaymentNotification) {
	// Initialize bot for sending notifications
	cfg := config.LoadConfig()
	bot, err := tgbotapi.NewBotAPI(cfg.ApiTGBotToken)
	if err != nil {
		log.Fatalf("Error creating notification bot: %v", err)
		return
	}

	for notification := range notifications {
		// Send success message to user
		msg := tgbotapi.NewMessage(notification.ChatID,
			"✅ Ваш платеж успешно обработан! Спасибо за покупку.\n\n"+
				"Сумма: $"+string(notification.Amount)+"\n"+
				"Выбранная валютная пара: "+notification.SessionID)

		// Add any additional buttons or information here

		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Error sending payment notification: %v", err)
		}

		// You can add additional processing here if needed,
		// such as activating user services, etc.
	}
}
