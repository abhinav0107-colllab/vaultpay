package service

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrPaymentAlreadyDisputed = errors.New("payment is already locked in an active dispute case")
	ErrPaymentNotFound        = errors.New("target payment transaction reference does not exist")
)

type Dispute struct {
	ID        string    `json:"id"`
	PaymentID string    `json:"payment_id"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"` // "OPEN", "RESOLVED_MERCHANT", "RESOLVED_CUSTOMER"
	CreatedAt time.Time `json:"created_at"`
}

type DisputeService struct {
	disputesMap sync.Map
	// Leverage our existing memory architecture simulation layers
	paymentsMap sync.Map
}

func NewDisputeService() *DisputeService {
	s := &DisputeService{}
	// Seed a sample transaction reference ID to test our webhook transitions smoothly
	s.paymentsMap.Store("ch_test_99", "SUCCESS")
	return s
}

// CreateDispute transitions an active payment to DISPUTED status and locks the funds
func (s *DisputeService) CreateDispute(id, paymentID, reason string) (*Dispute, error) {
	status, exists := s.paymentsMap.Load(paymentID)
	if !exists {
		return nil, ErrPaymentNotFound
	}

	if status.(string) == "DISPUTED" {
		return nil, ErrPaymentAlreadyDisputed
	}

	// Lock the target transaction record status securely
	s.paymentsMap.Store(paymentID, "DISPUTED")

	dispute := &Dispute{
		ID:        id,
		PaymentID: paymentID,
		Reason:    reason,
		Status:    "OPEN",
		CreatedAt: time.Now(),
	}

	s.disputesMap.Store(id, dispute)
	return dispute, nil
}
