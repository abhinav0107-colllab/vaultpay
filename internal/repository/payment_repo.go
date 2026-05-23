package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Payment struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"` // Measured in smaller subunits (Paise)
	Currency  string    `json:"currency"`
	Status    string    `json:"status"` // "PENDING", "COMPLETED", "FAILED"
	CreatedAt time.Time `json:"created_at"`
}

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// CreateTransactionRecord applies an atomic database ledger change safely
func (r *PaymentRepository) CreateTransactionRecord(ctx context.Context, userID string, amount int64, currency string) (*Payment, error) {
	// 1. Open a safe SQL Transaction isolation block
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open ledger transaction: %w", err)
	}

	// Rule: Defer a rollback. If the function exits early due to an error,
	// any partial database state adjustments are completely wiped clean.
	defer tx.Rollback()

	// 2. Fetch current balance and lock the row for update to prevent concurrent race conditions
	var currentBalance int64
	balanceQuery := `SELECT balance FROM users WHERE id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, balanceQuery, userID).Scan(&currentBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to look up target user balance: %w", err)
	}

	// 3. Prevent overdraft limits
	if currentBalance < amount {
		return nil, fmt.Errorf("insufficient account balances to clear transaction")
	}

	// 4. Update user account balance
	updateUserQuery := `UPDATE users SET balance = balance - $1 WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateUserQuery, amount, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust user ledger balance accounts: %w", err)
	}

	// 5. Insert audit log tracking record into payments
	insertPaymentQuery := `
		INSERT INTO payments (user_id, amount, currency, status)
		VALUES ($1, $2, $3, 'COMPLETED')
		RETURNING id, user_id, amount, currency, status, created_at`

	p := &Payment{}
	err = tx.QueryRowContext(ctx, insertPaymentQuery, userID, amount, currency).
		Scan(&p.ID, &p.UserID, &p.Amount, &p.Currency, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to register permanent payment record log: %w", err)
	}

	// 6. Commit transaction safely to the database disk
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit financial block adjustments: %w", err)
	}

	return p, nil
}
