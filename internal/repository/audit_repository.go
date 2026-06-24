package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type PaymentEvent struct {
	ID        int    `json:"id"`
	PaymentID string `json:"payment_id"`
	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
	CreatedAt string `json:"created_at"`
}

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// LogEvent appends a new immutable state transition record into our audit ledger
func (r *AuditRepository) LogEvent(paymentID, eventType string, payloadData interface{}) error {
	jsonPayload, err := json.Marshal(payloadData)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	query := `INSERT INTO payment_events (payment_id, event_type, payload) VALUES ($1, $2, $3)`
	_, err = r.db.Exec(query, paymentID, eventType, jsonPayload)
	return err
}

// ReplayHistory fetches every event for a payment order sequentially from start to finish
func (r *AuditRepository) ReplayHistory(paymentID string) ([]PaymentEvent, error) {
	query := `
		SELECT id, payment_id, event_type, payload, created_at 
		FROM payment_events 
		WHERE payment_id = $1 
		ORDER BY id ASC`

	rows, err := r.db.Query(query, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []PaymentEvent
	for rows.Next() {
		var e PaymentEvent
		if err := rows.Scan(&e.ID, &e.PaymentID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, e)
	}
	return history, nil
}

type StateTransitionAnalysis struct {
	ID             int    `json:"event_id"`
	EventType      string `json:"event_type"`
	CreatedAt      string `json:"changed_at"`
	SequenceNumber int    `json:"sequence_number"`
	PreviousState  string `json:"previous_state"`
}

// AnalyzeTransitions uses Window functions to map state progression sequences
func (r *AuditRepository) AnalyzeTransitions(paymentID string) ([]StateTransitionAnalysis, error) {
	query := `
		SELECT 
			id, 
			event_type, 
			created_at::text,
			ROW_NUMBER() OVER (PARTITION BY payment_id ORDER BY id ASC) as sequence_number,
			LAG(event_type, 1, 'START') OVER (PARTITION BY payment_id ORDER BY id ASC) as previous_state
		FROM payment_events
		WHERE payment_id = $1`

	rows, err := r.db.Query(query, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analysis []StateTransitionAnalysis
	for rows.Next() {
		var a StateTransitionAnalysis
		if err := rows.Scan(&a.ID, &a.EventType, &a.CreatedAt, &a.SequenceNumber, &a.PreviousState); err != nil {
			return nil, err
		}
		analysis = append(analysis, a)
	}
	return analysis, nil
}
