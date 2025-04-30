package payment

import (
	"Trading-Bot/config"
	"Trading-Bot/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
	"io"
	"log"
	"net/http"
	"strconv"
)

// StripeService handles all Stripe-related operations
type StripeService struct {
	apiKey          string
	webhookSecret   string
	db              *sqlx.DB
	notificationCh  chan models.PaymentNotification
	priceMap        map[string]string
	successRedirect string
	cancelRedirect  string
}

// NewStripeService creates a new stripe service
func NewStripeService(cfg *config.Config, db *sqlx.DB, notificationCh chan models.PaymentNotification) *StripeService {
	// Initialize Stripe with API key
	stripe.Key = cfg.StripeApiKey

	// Default price map - you can customize this based on your needs
	priceMap := map[string]string{
		"default": cfg.PriceID,
	}

	return &StripeService{
		apiKey:          cfg.StripeApiKey,
		webhookSecret:   cfg.StripeWebHook,
		db:              db,
		notificationCh:  notificationCh,
		priceMap:        priceMap,
		successRedirect: "https://t.me/your_bot_username", // Replace with your bot username
		cancelRedirect:  "https://t.me/your_bot_username", // Replace with your bot username
	}
}

// InitDatabase sets up the required database tables
func (s *StripeService) InitDatabase() error {
	// Create payments table if it doesn't exist
	_, err := s.db.Exec(`
        CREATE TABLE IF NOT EXISTS payments (
            id VARCHAR PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id BIGINT REFERENCES users_list(id),
            status VARCHAR NOT NULL CHECK (status IN ('pending', 'completed', 'failed')),
            session_id VARCHAR NOT NULL UNIQUE,
            amount BIGINT NOT NULL DEFAULT 0,
            created_at BIGINT NOT NULL DEFAULT extract(epoch from now()),
            completed_at BIGINT
        )
    `)
	return err
}

// CreateCheckoutSession creates a Stripe checkout session for the user
// CreateCheckoutSession creates a Stripe checkout session for the user
// CreateCheckoutSession creates a Stripe checkout session for the user
func (s *StripeService) CreateCheckoutSession(email string, currencyPair string, chatID int64, userID int64) (string, error) {
	if s.apiKey == "" {
		return "", errors.New("stripe API key is not set")
	}

	// Create metadata for the session
	metadata := map[string]string{
		"email":         email,
		"currency_pair": currencyPair,
		"chat_id":       strconv.FormatInt(chatID, 10),
		"user_id":       strconv.FormatInt(userID, 10),
	}

	// Create the checkout session with dynamic pricing
	params := &stripe.CheckoutSessionParams{
		Mode:          stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:    stripe.String(s.successRedirect),
		CancelURL:     stripe.String(s.cancelRedirect),
		CustomerEmail: stripe.String(email),
		Metadata:      metadata,
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(fmt.Sprintf("Trading Bot Subscription for %s", currencyPair)),
					},
					UnitAmount: stripe.Int64(900), // $9.00 in cents
				},
				Quantity: stripe.Int64(1),
			},
		},
	}

	// Create the session
	session, err := session.New(params)
	if err != nil {
		log.Printf("Error creating checkout session: %v", err)
		return "", err
	}

	// Record the payment in pending status
	if err := s.recordPayment(session.ID, userID, "pending", session.AmountTotal); err != nil {
		log.Printf("Warning: Failed to record pending payment: %v", err)
		// Continue despite the error to allow payment to proceed
	}

	return session.URL, nil
}

// HandleWebhook processes Stripe webhook events
func (s *StripeService) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request", http.StatusInternalServerError)
		return
	}

	// Get the signature from headers
	stripeSignature := r.Header.Get("Stripe-Signature")

	// Verify signature
	event, err := webhook.ConstructEvent(payload, stripeSignature, s.webhookSecret)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	// Process different event types
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Printf("Error parsing session data: %v", err)
			http.Error(w, "Error parsing data", http.StatusInternalServerError)
			return
		}

		// Get user information from metadata
		email := session.Metadata["email"]
		chatIDStr := session.Metadata["chat_id"]
		userIDStr := session.Metadata["user_id"]

		if email == "" || chatIDStr == "" || userIDStr == "" {
			log.Printf("Incomplete session metadata: email=%s, chat_id=%s, user_id=%s",
				email, chatIDStr, userIDStr)
			http.Error(w, "Incomplete metadata", http.StatusBadRequest)
			return
		}

		// Convert chatID and userID to int64
		chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
		if err != nil {
			log.Printf("Error parsing chat ID: %v", err)
			http.Error(w, "Invalid chat ID", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			log.Printf("Error parsing user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Update payment status in the database
		if err := s.recordPayment(session.ID, userID, "completed", session.AmountTotal); err != nil {
			log.Printf("Error updating payment record: %v", err)
			// Continue despite the error to notify the user
		}

		// Send notification about successful payment
		notification := models.PaymentNotification{
			ChatID:    chatID,
			Email:     email,
			SessionID: session.ID,
			Status:    "completed",
			Amount:    session.AmountTotal / 100, // Convert cents to dollars/euros
		}
		s.notificationCh <- notification

		log.Printf("Payment successfully processed for %s", email)
	}

	w.WriteHeader(http.StatusOK)
}

// recordPayment inserts or updates a payment record in the database
func (s *StripeService) recordPayment(sessionID string, userID int64, status string, amount int64) error {
	// First, validate that the status is one of the allowed values
	if status != "pending" && status != "completed" && status != "failed" {
		return fmt.Errorf("invalid payment status: %s", status)
	}

	query := `
        INSERT INTO payments (
            user_id, 
            status, 
            session_id, 
            amount
            -- created_at and completed_at use DEFAULT values
        ) VALUES (
            $1, 
            $2, 
            $3, 
            $4
        )
        ON CONFLICT (session_id) DO UPDATE SET
            status = EXCLUDED.status,
            completed_at = CASE WHEN EXCLUDED.status = 'completed' THEN extract(epoch from now()) ELSE payments.completed_at END`

	_, err := s.db.Exec(query, userID, status, sessionID, amount)
	if err != nil {
		log.Printf("Error recording payment: %v", err)
		return err
	}

	// If payment is completed, update the user's paid status
	if status == "completed" {
		updateUserQuery := `UPDATE users_list SET paid = TRUE WHERE id = $1`
		_, err = s.db.Exec(updateUserQuery, userID)
		if err != nil {
			log.Printf("Error updating user paid status: %v", err)
			return err
		}
		log.Printf("User paid status updated for UserID=%d", userID)
	}

	log.Printf("Payment recorded: UserID=%d, SessionID=%s, Status=%s", userID, sessionID, status)
	return nil
}

// StartWebhookServer starts the HTTP server for handling Stripe webhooks
func (s *StripeService) StartWebhookServer(port string) {
	// Set up routes
	http.HandleFunc("/webhook", s.HandleWebhook)

	// Start server
	addr := ":" + port
	log.Printf("Starting webhook server on %s", addr)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Failed to start webhook server: %v", err)
		}
	}()
}
