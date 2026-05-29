package tasks

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// Define our unique distributed task queue identifier name
const TypeWebhookRetryEvent = "webhook:retry_dispatch"

// WebhookPayload represents the data structure saved inside Redis for processing
type WebhookPayload struct {
	PaymentID   string `json:"payment_id"`
	TargetURL   string `json:"target_url"`
	EventStatus string `json:"event_status"`
}

// NewWebhookDispatchTask packs a webhook payload into an executable Asynq task element
func NewWebhookDispatchTask(paymentID, url, status string) (*asynq.Task, error) {
	payload := WebhookPayload{
		PaymentID:   paymentID,
		TargetURL:   url,
		EventStatus: status,
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize task parameters: %w", err)
	}

	// Distribute this task with an explicit payload signature blueprint
	return asynq.NewTask(TypeWebhookRetryEvent, bytes), nil
}
