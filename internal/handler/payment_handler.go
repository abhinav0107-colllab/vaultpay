package handler

import (
	"encoding/json"
	"net/http"

	"github.com/abhinav0107-collab/vaultpay/internal/service"

	// 🔥 THESE IMPORTS REMOVE THE RED SQUIGGLES UNDER OTEL AND ATTRIBUTE
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
}

func NewPaymentHandler(ps *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: ps}
}

type ChargeRequest struct {
	UserID   string `json:"user_id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type RefundRequest struct {
	PaymentID string `json:"payment_id"`
}

// Declare our OpenTelemetry system tracer handle
var tracer = otel.Tracer("payment-handler-tracer")

// CreateChargeHandler processes inbound payments with active OpenTelemetry tracing
func (h *PaymentHandler) CreateChargeHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Start the OpenTelemetry trace span as request enters the pipeline
	ctx, span := tracer.Start(r.Context(), "CreateChargeHandler")
	defer span.End()

	var req ChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err) // Log JSON syntax errors directly to the trace graph
		http.Error(w, "Malformed JSON layout string", http.StatusBadRequest)
		return
	}

	// Validation Check Block
	if req.UserID == "" {
		http.Error(w, "user_id is a mandatory required field", http.StatusUnprocessableEntity)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, "transaction amount must be strictly positive", http.StatusUnprocessableEntity)
		return
	}
	// 🔥 DAY 18 ENFORCEMENT: Validate currency support structures
	if req.Currency != "USD" && req.Currency != "INR" {
		span.SetAttributes(attribute.String("error.message", "Unsupported currency type: "+req.Currency))
		http.Error(w, "Invalid currency. Only USD and INR are supported.", http.StatusUnprocessableEntity)
		return
	}

	// 2. Add extra searchable meta-tags to your Jaeger dashboard panel
	span.SetAttributes(
		attribute.String("payment.user_id", req.UserID),
		attribute.Int64("payment.amount", req.Amount),
		attribute.String("payment.currency", req.Currency),
	)

	// 3. Pass the traced 'ctx' context down to the service layer chain
	payment, err := h.paymentService.ProcessCharge(ctx, req.UserID, req.Amount, req.Currency)
	if err != nil {
		span.RecordError(err) // Log processing errors directly to the trace graph
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payment)
}

// CreateRefundHandler handles transactional rollback request events
func (h *PaymentHandler) CreateRefundHandler(w http.ResponseWriter, r *http.Request) {
	var req RefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Malformed JSON layout payload string", http.StatusBadRequest)
		return
	}

	if req.PaymentID == "" {
		http.Error(w, "payment_id is a mandatory tracking field", http.StatusUnprocessableEntity)
		return
	}

	payment, err := h.paymentService.ProcessRefund(r.Context(), req.PaymentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payment)
}
