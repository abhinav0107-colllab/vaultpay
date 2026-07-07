package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/database"
)

type PaymentStatus string

type Payment struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TransactionQuery defines filters for cursor-based fetches

// ========================================================================
// 🛡️ INTERFACE DECLARATION FOR THE SERVICE LAYER
// ========================================================================
type PaymentRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateTransactionRecord(ctx context.Context, userID string, amount int64, currency string) (*Payment, error)
	CreateTransactionRecordTx(ctx context.Context, tx *sql.Tx, userID string, amount int64, currency string) (*Payment, error)
	RefundTransactionRecord(ctx context.Context, paymentID string) (*Payment, error)
	UpdatePaymentStatus(ctx context.Context, paymentID string, nextStatus string) error
	GetPaginatedTransactions(ctx context.Context, q TransactionQuery) ([]Transaction, string, error)
}

type sqlPaymentRepository struct {
	cluster *database.DatabaseCluster
}

// NewPaymentRepository instantiates our master-replica mapping layer matching the interface
func NewPaymentRepository(cluster *database.DatabaseCluster) PaymentRepository {
	return &sqlPaymentRepository{
		cluster: cluster,
	}
}

// ========================================================================
// ⚡ TRANSACTION PROPAGATION METRIC CORES (Day 34)
// ========================================================================

// BeginTx safely initiates an atomic transaction context bound explicitly to the MASTER engine pool
func (r *sqlPaymentRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	txOpts := &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}
	return r.cluster.Master.BeginTx(ctx, txOpts)
}

// CreateTransactionRecordTx processes ledger changes directly using an inherited transactional pointer
func (r *sqlPaymentRepository) CreateTransactionRecordTx(ctx context.Context, tx *sql.Tx, userID string, amount int64, currency string) (*Payment, error) {
	// 1. Fetch current balance and lock the row for update inside the existing tx frame
	var currentBalance int64
	balanceQuery := `SELECT balance FROM users WHERE id = $1 FOR UPDATE`
	err := tx.QueryRowContext(ctx, balanceQuery, userID).Scan(&currentBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to look up target user balance inside tx: %w", err)
	}

	// 2. Prevent overdraft limits
	if currentBalance < amount {
		return nil, fmt.Errorf("insufficient account balances to clear transaction")
	}

	// 3. Update user account balance safely
	updateUserQuery := `UPDATE users SET balance = balance - $1 WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateUserQuery, amount, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust user ledger balance accounts: %w", err)
	}

	// 4. Insert database record ledger tracking status
	insertPaymentQuery := `
		INSERT INTO payments (user_id, amount, currency, status)
		VALUES ($1, $2, $3, 'succeeded')
		RETURNING id, user_id, amount, currency, status, created_at`

	p := &Payment{}
	err = tx.QueryRowContext(ctx, insertPaymentQuery, userID, amount, currency).Scan(
		&p.ID, &p.UserID, &p.Amount, &p.Currency, &p.Status, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register permanent payment record log: %w", err)
	}

	return p, nil
}

// ========================================================================
// 📦 STANDALONE ORIGINAL REPOSITORY METHOD CONTRACTS
// ========================================================================

// CreateTransactionRecord applies an independent atomic database ledger change safely
func (r *sqlPaymentRepository) CreateTransactionRecord(ctx context.Context, userID string, amount int64, currency string) (*Payment, error) {
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	p, err := r.CreateTransactionRecordTx(ctx, tx, userID, amount, currency)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit financial block adjustments: %w", err)
	}

	return p, nil
}

// RefundTransactionRecord safely processes a refund event, validating state machine rules
func (r *sqlPaymentRepository) RefundTransactionRecord(ctx context.Context, paymentID string) (*Payment, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	tx, err := r.cluster.Master.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open refund transaction: %w", err)
	}
	defer tx.Rollback()

	var p Payment
	query := `
		SELECT id, user_id, amount, currency, status, created_at, updated_at 
		FROM payments 
		WHERE id = $1 FOR UPDATE`

	err = tx.QueryRowContext(ctx, query, paymentID).Scan(&p.ID, &p.UserID, &p.Amount, &p.Currency, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("payment record not found or lock acquisition timed out: %w", err)
	}
	if p.Status != "succeeded" && p.Status != "paid" && p.Status != "COMPLETED" {
		return nil, fmt.Errorf("invalid state transition: cannot refund a transaction with status '%s'", p.Status)
	}

	updateUserQuery := `UPDATE users SET balance = balance + $1 WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateUserQuery, p.Amount, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to return funds to user wallet balance: %w", err)
	}

	updatePaymentQuery := `
		UPDATE payments 
		SET status = 'refunded', updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1 
		RETURNING status, updated_at`

	err = tx.QueryRowContext(ctx, updatePaymentQuery, paymentID).Scan(&p.Status, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment record lifecycle status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit financial refund block: %w", err)
	}

	return &p, nil
}

// UpdatePaymentStatus updates the status of a specific payment tracking record
func (r *sqlPaymentRepository) UpdatePaymentStatus(ctx context.Context, paymentID string, nextStatus string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		UPDATE payments 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2`

	_, err := r.cluster.Master.ExecContext(ctx, query, nextStatus, paymentID)
	return err
}

// GetPaginatedTransactions fetches records using high-performance cursor-based pagination
func (r *sqlPaymentRepository) GetPaginatedTransactions(ctx context.Context, q TransactionQuery) ([]Transaction, string, error) {
	query := `
		SELECT id::text, amount, currency, status, created_at 
		FROM payments 
		%s 
		ORDER BY id DESC 
		LIMIT $1`

	var rows *sql.Rows
	var err error

	if q.NextCursor != "" {
		filterClause := "WHERE id::text < $2"
		sqlQuery := fmt.Sprintf(query, filterClause)
		rows, err = r.cluster.Replica.QueryContext(ctx, sqlQuery, q.Limit, q.NextCursor)
	} else {
		sqlQuery := fmt.Sprintf(query, "")
		rows, err = r.cluster.Replica.QueryContext(ctx, sqlQuery, q.Limit)
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

	nextPageCursor := ""
	if len(transactions) == q.Limit {
		nextPageCursor = lastID
	}

	return transactions, nextPageCursor, nil
}
