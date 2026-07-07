package service

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/database"
	"github.com/abhinav0107-collab/vaultpay/internal/tasks"
	"github.com/hibiken/asynq"
)

type OutboxWorker struct {
	dbCluster    *database.DatabaseCluster
	asynqClient  *asynq.Client
	pollInterval time.Duration
	stopChan     chan struct{}
}

func NewOutboxWorker(cluster *database.DatabaseCluster, client *asynq.Client, interval time.Duration) *OutboxWorker {
	return &OutboxWorker{
		dbCluster:    cluster,
		asynqClient:  client,
		pollInterval: interval,
		stopChan:     make(chan struct{}),
	}
}

func (w *OutboxWorker) Start() {
	ticker := time.NewTicker(w.pollInterval)
	log.Printf("📥 Transactional Outbox Poller active (Interval: %v)", w.pollInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				w.ProcessPendingEvents(context.Background())
			case <-w.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (w *OutboxWorker) Stop() {
	close(w.stopChan)
}

// ProcessPendingEvents queries, dispatches, and updates staged events atomically
func (w *OutboxWorker) ProcessPendingEvents(ctx context.Context) {
	// 1. Lock a batch of pending events using SKIP LOCKED to ensure concurrent safety
	tx, err := w.dbCluster.Master.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ Outbox Worker: failed to start transaction: %v", err)
		return
	}
	defer tx.Rollback()

	query := `
		SELECT id, event_type, payload 
		FROM outbox_events 
		WHERE status = 'PENDING' 
		ORDER BY created_at ASC 
		LIMIT 10 
		FOR UPDATE SKIP LOCKED`

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		log.Printf("❌ Outbox Worker: failed to fetch pending events: %v", err)
		return
	}
	defer rows.Close()

	type fetchedEvent struct {
		ID        string
		EventType string
		Payload   []byte
	}

	var events []fetchedEvent
	for rows.Next() {
		var ev fetchedEvent
		if err := rows.Scan(&ev.ID, &ev.EventType, &ev.Payload); err != nil {
			log.Printf("❌ Outbox Worker: row scan error: %v", err)
			continue
		}
		events = append(events, ev)
	}
	rows.Close() // Close early to prepare for execution modifications

	if len(events) == 0 {
		return // No pending work to do
	}

	// 2. Iterate through events, dispatch to Asynq background workers, and mark as processed
	updateQuery := `UPDATE outbox_events SET status = $1, processed_at = $2 WHERE id = $3`

	for _, ev := range events {
		// Instantiate an Asynq background execution task mapping to your webhook pipeline
		task := asynq.NewTask(tasks.TypeWebhookRetryEvent, ev.Payload)

		// Enqueue into Asynq queue
		_, err = w.asynqClient.EnqueueContext(ctx, task)
		status := "PROCESSED"
		var processedAt sql.NullTime
		processedAt.Time = time.Now()
		processedAt.Valid = true

		if err != nil {
			log.Printf("⚠️ Outbox Worker: failed to enqueue task %s to Asynq: %v", ev.ID, err)
			status = "FAILED"
			processedAt.Valid = false // Keep trying later or flag for manual review
		}

		_, err = tx.ExecContext(ctx, updateQuery, status, processedAt, ev.ID)
		if err != nil {
			log.Printf("❌ Outbox Worker: failed to update event status %s: %v", ev.ID, err)
			return // Break transaction out safely on systemic db failure
		}
	}

	// 3. Commit the batch transaction cleanly
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Outbox Worker: batch transaction commit failed: %v", err)
	}
}
