package repository

import (
	"context"
	"database/sql"
)

type PostgresPaymentEventRepository struct {
	db *sql.DB
}

func NewPostgresPaymentEventRepository(db *sql.DB) *PostgresPaymentEventRepository {
	return &PostgresPaymentEventRepository{db: db}
}

func (r *PostgresPaymentEventRepository) CreateEvent(ctx context.Context, event *PaymentAuditLog) error {
	query := `
		INSERT INTO payment_events (payment_id, event_type, previous_state, current_state, payload)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query,
		event.PaymentID,
		event.EventType,
		event.PreviousState,
		event.CurrentState,
		event.Payload,
	).Scan(&event.ID, &event.CreatedAt)
}

func (r *PostgresPaymentEventRepository) GetHistoryByPaymentID(ctx context.Context, paymentID string) ([]PaymentAuditLog, error) {
	query := `
		SELECT id, payment_id, event_type, previous_state, current_state, payload, created_at
		FROM payment_events
		WHERE payment_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []PaymentAuditLog
	for rows.Next() {
		var e PaymentAuditLog
		err := rows.Scan(&e.ID, &e.PaymentID, &e.EventType, &e.PreviousState, &e.CurrentState, &e.Payload, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}
