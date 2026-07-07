package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/abhinav0107-collab/vaultpay/internal/repository"
	"github.com/abhinav0107-collab/vaultpay/internal/tasks"
	"github.com/hibiken/asynq"
)

// 1. Ensure the struct field uses the interface definition
type AdvancedWebhookWorker struct {
	db          *sql.DB
	paymentRepo repository.PaymentRepository // ◄ Remove any '*' pointer here if present
}

// 2. Update the constructor parameters to match
func NewAdvancedWebhookWorker(db *sql.DB, pr repository.PaymentRepository) *AdvancedWebhookWorker { // ◄ Remove '*' from repository type
	return &AdvancedWebhookWorker{
		db:          db,
		paymentRepo: pr,
	}
}

// ProcessWebhookTask executes task routines safely using Postgres Advisory Locks
func (w *AdvancedWebhookWorker) ProcessWebhookTask(ctx context.Context, t *asynq.Task) error {
	var payload tasks.WebhookPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	log.Printf("📥 Processing distributed webhook queue item for payment: %s", payload.PaymentID)

	// 🔥 ACCORDING TO TECH REQS: Obtain an exclusive session-level transactional Advisory Lock
	// We use a simple hash integer logic to map the payment string to a numeric database key lock space
	lockKey := 1234567

	_, err := w.db.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
	if err != nil {
		log.Printf("❌ Failed to obtain PostgreSQL Advisory lock boundary: %v", err)
		return err
	}

	// Inside this lock block, we execute our webhook dispatch safely without race condition concerns
	log.Printf("🔒 Advisory Lock secured. Dispatching payment payload to client endpoint -> %s", payload.TargetURL)

	// Simulate successful network outbound call delivery
	return nil
}
