package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type OutboxEvent struct {
	ID            string     `db:"id"`
	AggregateType string     `db:"aggregate_type"`
	AggregateID   string     `db:"aggregate_id"`
	EventType     string     `db:"event_type"`
	Payload       string     `db:"payload"` // JSONB string
	Status        string     `db:"status"`  // PENDING, PROCESSED, FAILED
	RetryCount    int        `db:"retry_count"`
	CreatedAt     time.Time  `db:"created_at"`
	ProcessedAt   *time.Time `db:"processed_at"`
}

type OutboxRepository interface {
	SaveEventTx(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload interface{}) error
}

type sqlOutboxRepository struct{}

func NewOutboxRepository() OutboxRepository {
	return &sqlOutboxRepository{}
}

// SaveEventTx writes the outbox event using an existing transactional block context
func (r *sqlOutboxRepository) SaveEventTx(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload interface{}) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload, status)
		VALUES ($1, $2, $3, $4, 'PENDING')
	`

	_, err = tx.ExecContext(ctx, query, aggregateType, aggregateID, eventType, bytes)
	return err
}
