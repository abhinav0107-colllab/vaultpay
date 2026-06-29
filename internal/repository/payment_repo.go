package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type PaymentStatus string

type Payment struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"` // <-- ADD THIS LINE
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

// RefundTransactionRecord safely processes a refund event, validating the state machine rules
func (r *PaymentRepository) RefundTransactionRecord(ctx context.Context, paymentID string) (*Payment, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open refund transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Fetch current transaction status and merchant information under an exclusive FOR UPDATE lock
	var p Payment
	query := `
		SELECT id, user_id, amount, currency, status, created_at, updated_at 
		FROM payments 
		WHERE id = $1 FOR UPDATE`

	err = tx.QueryRowContext(ctx, query, paymentID).Scan(&p.ID, &p.UserID, &p.Amount, &p.Currency, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("payment record not found or lock acquisition timed out: %w", err)
	}
	if p.Status != "succeeded" && p.Status != "paid" {
		return nil, fmt.Errorf("invalid state transition: cannot refund a transaction with status '%s'", p.Status)
	}
	updateUserQuery := `UPDATE users SET balance = balance + $1 WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateUserQuery, p.Amount, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to return funds to user wallet balance: %w", err)
	}

	// 4. Update the payment status to 'refunded' inside our system matrix
	updatePaymentQuery := `
		UPDATE payments 
		SET status = 'refunded', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1 
		RETURNING status, updated_at`

	err = tx.QueryRowContext(ctx, updatePaymentQuery, paymentID).Scan(&p.Status, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment record lifecycle status: %w", err)
	}

	// 5. Securely commit our balance restorations onto the physical disk
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit financial refund block: %w", err)
	}

	return &p, nil
}

// UpdatePaymentStatus updates the status of a specific payment tracking record
func (r *PaymentRepository) UpdatePaymentStatus(ctx context.Context, paymentID string, nextStatus string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		UPDATE payments 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, nextStatus, paymentID)
	return err
}

// GetPaginatedTransactions fetches records using high-performance cursor-based pagination
func (r *PaymentRepository) GetPaginatedTransactions(ctx context.Context, q TransactionQuery) ([]Transaction, string, error) {
	// 1. Base query structure sorting chronologically backwards (newest first)
	query := `
		SELECT id::text, amount, currency, status, created_at 
		FROM payments 
		%s 
		ORDER BY id DESC 
		LIMIT $1`

	var rows *sql.Rows
	var err error

	// 2. Dynamically apply the cursor filter pointer if it exists
	if q.NextCursor != "" {
		// Since payments uses UUID/Serial IDs, we use standard string matching evaluation
		filterClause := "WHERE id::text < $2"
		sqlQuery := fmt.Sprintf(query, filterClause)
		rows, err = r.db.QueryContext(ctx, sqlQuery, q.Limit, q.NextCursor)
	} else {
		sqlQuery := fmt.Sprintf(query, "")
		rows, err = r.db.QueryContext(ctx, sqlQuery, q.Limit)
	}

	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var transactions []Transaction
	var lastID string

	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Currency, &t.Status, &t.CreatedAt); err != nil {
			return nil, "", err
		}
		transactions = append(transactions, t)
		lastID = t.ID
	}

	// 3. If we hit our full page limit capacity, pass back the last item's ID as the next page pointer
	nextPageCursor := ""
	if len(transactions) == q.Limit {
		nextPageCursor = lastID
	}

	return transactions, nextPageCursor, nil
}
