package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
)

// 1. Update the interface to accept all 5 parameters matching your database schema
type SubscriptionService interface {
	CreateSubscription(id string, userID string, planID string, amount int64, period string) (interface{}, error)
}

type SubscriptionHandler struct {
	subService SubscriptionService
}

// 2. Accept the local interface type 'SubscriptionService' directly
func NewSubscriptionHandler(s SubscriptionService) *SubscriptionHandler { // ◄ Removed 'service.' prefix
	return &SubscriptionHandler{subService: s}
}

type SubscriptionRequest struct {
	UserID        string `json:"user_id"`
	PlanID        string `json:"plan_id"`
	Amount        int64  `json:"amount"`
	BillingPeriod string `json:"billing_period"`
}

// AFTER (Update the receiver function signature to match):
func (h *SubscriptionHandler) CreateSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var req SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteProblemResponse(w, r.URL.Path, http.StatusBadRequest, "Malformed Request Payload", "The JSON syntax provided in the request body is invalid.")
		return
	}

	// 1. Generate a unique subscription tracking identifier securely
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	subID := fmt.Sprintf("sub_%x", b)

	// 2. Call the backend domain service layer passing all 5 arguments
	sub, err := h.subService.CreateSubscription(subID, req.UserID, req.PlanID, req.Amount, req.BillingPeriod)
	if err != nil {
		WriteProblemResponse(w, r.URL.Path, http.StatusUnprocessableEntity, "Subscription Processing Failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sub)
}
