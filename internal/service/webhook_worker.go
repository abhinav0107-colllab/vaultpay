package service

import (
	"context"
	"log"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/repository"
)

type WebhookWorker struct {
	paymentRepo *repository.PaymentRepository
	secretKey   string
}

func NewWebhookWorker(pr *repository.PaymentRepository, secret string) *WebhookWorker {
	return &WebhookWorker{
		paymentRepo: pr,
		secretKey:   secret,
	}
}

// StartWorkerEngine launches a background concurrency loop running alongside the main app
func (w *WebhookWorker) StartWorkerEngine(ctx context.Context) {
	log.Println("String engine execution loop...")
	log.Println("⚡ VaultPay Background Webhook Dispatch Engine is live...")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Shutting down background webhook worker safely...")
			return
		case <-ticker.C:
			// The background loop ticks every 5 seconds to process queue stages asynchronously
		}
	}
}
