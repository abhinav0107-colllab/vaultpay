package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/abhinav0107-collab/vaultpay/internal/service"
)

type SubscriptionHandler struct {
	subService *service.SubscriptionService
}

func NewSubscriptionHandler(s *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subService: s}
}

type SubscriptionRequest struct {
	UserID        string `json:"user_id"`
	PlanID        string `json:"plan_id"`
	Amount        int64  `json:"amount"`
	BillingPeriod string `json:"billing_period"`
}

func (h *SubscriptionHandler) CreateSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var req SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteProblemResponse(w, r.URL.Path, http.StatusBadRequest, "Malformed Request Payload", "The JSON syntax provided in the request body is invalid or cannot be parsed.")
		return
	}

	// Generate a unique subscription tracking identifier securely
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	subID := fmt.Sprintf("sub_%x", b)

	sub, err := h.subService.CreateSubscription(subID, req.UserID, req.PlanID, req.Amount, req.BillingPeriod)
	if err != nil {
		WriteProblemResponse(w, r.URL.Path, http.StatusUnprocessableEntity, "Subscription Processing Failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sub)
}
