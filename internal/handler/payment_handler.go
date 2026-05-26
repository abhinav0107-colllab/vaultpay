package handler

import (
	"encoding/json"
	"net/http"

	"github.com/abhinav0107-collab/vaultpay/internal/service"
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

func (h *PaymentHandler) CreateChargeHandler(w http.ResponseWriter, r *http.Request) {
	var req ChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Malformed JSON layout string", http.StatusBadRequest)
		return
	}

	// 🔥 VALIDATION CHECK BLOCK START
	if req.UserID == "" {
		http.Error(w, "user_id is a mandatory required field", http.StatusUnprocessableEntity)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "transaction amount must be strictly positive", http.StatusUnprocessableEntity)
		return
	}
	// 🔥 VALIDATION CHECK BLOCK END

	payment, err := h.paymentService.ProcessCharge(r.Context(), req.UserID, req.Amount, req.Currency)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payment)
}

type RefundRequest struct {
	PaymentID string `json:"payment_id"`
}

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
