package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type PostgresSubscriptionRepository struct {
	db *sql.DB
}

func NewPostgresSubscriptionRepository(db *sql.DB) *PostgresSubscriptionRepository {
	return &PostgresSubscriptionRepository{db: db}
}

func (r *PostgresSubscriptionRepository) Create(ctx context.Context, sub *Subscription) error {
	query := `
		INSERT INTO subscriptions (merchant_id, plan_name, billing_period, status, current_period_start, current_period_end)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		sub.MerchantID,
		sub.PlanName,
		sub.BillingPeriod,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

func (r *PostgresSubscriptionRepository) GetByMerchantID(ctx context.Context, merchantID string) (*Subscription, error) {
	query := `
		SELECT id, merchant_id, plan_name, billing_period, status, current_period_start, current_period_end, created_at, updated_at
		FROM subscriptions
		WHERE merchant_id = $1`

	var sub Subscription
	err := r.db.QueryRowContext(ctx, query, merchantID).Scan(
		&sub.ID, &sub.MerchantID, &sub.PlanName, &sub.BillingPeriod,
		&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No active subscription found safely
		}
		return nil, err
	}
	return &sub, nil
}

func (r *PostgresSubscriptionRepository) Update(ctx context.Context, sub *Subscription) error {
	query := `
		UPDATE subscriptions
		SET plan_name = $1, billing_period = $2, status = $3, current_period_start = $4, current_period_end = $5, updated_at = NOW()
		WHERE id = $6`

	_, err := r.db.ExecContext(ctx, query, sub.PlanName, sub.BillingPeriod, sub.Status, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.ID)
	return err
}

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
