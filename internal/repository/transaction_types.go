package repository

import (
	"time"
)

// Transaction represents a historical financial record in VaultPay
type Transaction struct {
	ID        string    `json:"id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// TransactionQuery defines the structural limits and cursor pointer values for pagination
type TransactionQuery struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor"` // Holds the ID pointer where the last page left off
}
