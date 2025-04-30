package models

// Payment represents a payment record in the database
type Payment struct {
	ID          string `db:"id"`
	UserID      int64  `db:"user_id"`
	Status      string `db:"status"`
	SessionID   string `db:"session_id"`
	Amount      int64  `db:"amount"`
	CreatedAt   int64  `db:"created_at"`
	CompletedAt int64  `db:"completed_at"`
}

// PaymentNotification is used to communicate payment status changes
type PaymentNotification struct {
	ChatID    int64
	Email     string
	SessionID string
	Status    string
	Amount    int64
}
type User struct {
	ID           int64  `db:"id"`
	Email        string `db:"email"`
	CurrencyPair string `db:"currency_pair"`
	Price        int    `db:"price"`
	ChatID       int64  `db:"chat_id"`
	Paid         bool   `db:"paid"`
}
