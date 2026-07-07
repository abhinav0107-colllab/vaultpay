package service

import (
	"context"
	"fmt"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/repository"
)

type PaymentService struct {
	paymentRepo repository.PaymentRepository
	outboxRepo  repository.OutboxRepository // ◄ Outbox dependency field link
}

func NewPaymentService(pr repository.PaymentRepository, or repository.OutboxRepository) *PaymentService {
	return &PaymentService{
		paymentRepo: pr,
		outboxRepo:  or,
	}
}

// ProcessCharge processes inbound payments atomically along with an Outbox Event capture
func (s *PaymentService) ProcessCharge(ctx context.Context, userID string, amount int64, currency string) (*repository.Payment, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("transaction amount must be strictly positive")
	}

	// 1. Open up a strict atomic database transaction block context
	tx, err := s.paymentRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize payment transaction context: %w", err)
	}

	// Safely trigger a rollback fallback loop if any execution step fails mid-flight
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 2. Write the transaction entry using the active transaction pointer (tx)
	payment, err := s.paymentRepo.CreateTransactionRecordTx(ctx, tx, userID, amount, currency)
	if err != nil {
		return nil, err
	}

	// 3. Assemble the payload metadata exactly as the downstream merchant's webhook expects it
	webhookPayload := map[string]interface{}{
		"id":         payment.ID,
		"user_id":    payment.UserID,
		"amount":     payment.Amount,
		"currency":   payment.Currency,
		"status":     "payment.succeeded",
		"created_at": time.Now().Unix(),
	}

	// 4. Record the transactional outbox payload event within the SAME atomic database frame
	err = s.outboxRepo.SaveEventTx(ctx, tx, "payment", payment.ID, "payment.succeeded", webhookPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to stage transactional outbox event: %w", err)
	}

	// 5. Commit everything to persistent disk arrays cleanly together
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit payment transaction chain: %w", err)
	}

	return payment, nil
}

// ProcessRefund handles transactional rollback request events
func (s *PaymentService) ProcessRefund(ctx context.Context, paymentID string) (*repository.Payment, error) {
	if paymentID == "" {
		return nil, fmt.Errorf("payment tracking reference ID cannot be empty")
	}
	return s.paymentRepo.RefundTransactionRecord(ctx, paymentID)
}
