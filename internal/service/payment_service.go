package service

import (
	"context"
	"fmt"

	"github.com/abhinav0107-collab/vaultpay/internal/repository"
)

type PaymentService struct {
	paymentRepo *repository.PaymentRepository
}

func NewPaymentService(pr *repository.PaymentRepository) *PaymentService {
	return &PaymentService{paymentRepo: pr}
}

func (s *PaymentService) ProcessCharge(ctx context.Context, userID string, amount int64, currency string) (*repository.Payment, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("transaction amount must be strictly positive")
	}
	return s.paymentRepo.CreateTransactionRecord(ctx, userID, amount, currency)
}
