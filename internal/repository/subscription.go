package repository

import (
	"context"
	"time"
)

type Subscription struct {
	ID                 string    `json:"id"`
	MerchantID         string    `json:"merchant_id"`
	PlanName           string    `json:"plan_name"`
	BillingPeriod      string    `json:"billing_period"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SubscriptionRepository defines the database operations available
type SubscriptionRepository interface {
	Create(ctx context.Context, sub *Subscription) error
	GetByMerchantID(ctx context.Context, merchantID string) (*Subscription, error)
	Update(ctx context.Context, sub *Subscription) error
}