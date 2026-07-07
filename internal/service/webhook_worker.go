package service

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/repository"
	"github.com/hibiken/asynq"
)

type AdvancedWebhookWorker struct {
	db          *sql.DB
	paymentRepo repository.PaymentRepository
	httpClient  *http.Client
}

func NewAdvancedWebhookWorker(db *sql.DB, pr repository.PaymentRepository) *AdvancedWebhookWorker {
	return &AdvancedWebhookWorker{
		db:          db,
		paymentRepo: pr,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Always enforce timeouts for outbound client requests
		},
	}
}

// ProcessWebhookTask consumes events forwarded by the Outbox Poller from Asynq/Redis
func (w *AdvancedWebhookWorker) ProcessWebhookTask(ctx context.Context, t *asynq.Task) error {
	log.Printf("🚀 Asynq Worker: Processing webhook task event: %s", t.Type())

	// 1. In a production pipeline, your payload typically contains the target URL and metadata.
	// For testing purposes, we will mock sending this to a simulated merchant dashboard endpoint.
	targetMerchantURL := "http://httpbin.org/post"

	// 2. Dispatch the outbound POST request with the event payload
	req, err := http.NewRequestWithContext(ctx, "POST", targetMerchantURL, bytes.NewBuffer(t.Payload()))
	if err != nil {
		log.Printf("❌ Failed to construct outbound webhook request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "VaultPay-Webhook-Dispatcher/1.0")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		log.Printf("⚠️ Webhook Delivery Failed (Network Error): %v. Flagging for automatic Asynq retry...", err)
		// Returning an error automatically tells Asynq to kick off its built-in exponential backoff retry system!
		return fmt.Errorf("network delivery failure: %w", err)
	}
	defer resp.Body.Close()

	// 3. Evaluate the merchant response status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("⚠️ Merchant returned unhealthy status code: %d. Retrying...", resp.StatusCode)
		return fmt.Errorf("merchant unhealthy response status: %d", resp.StatusCode)
	}

	log.Printf("✅ Webhook delivered successfully to %s! Status: %s", targetMerchantURL, resp.Status)
	return nil
}
