package repository

import (
	"context"
	"encoding/json"
	"time"
)

type PaymentAuditLog struct {
	ID            string          `json:"id"`
	PaymentID     string          `json:"payment_id"`
	EventType     string          `json:"event_type"`
	PreviousState string          `json:"previous_state"`
	CurrentState  string          `json:"current_state"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
}

type PaymentEventRepository interface {
	CreateEvent(ctx context.Context, event *PaymentAuditLog) error
	GetHistoryByPaymentID(ctx context.Context, paymentID string) ([]PaymentAuditLog, error)
}
