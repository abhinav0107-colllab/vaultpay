package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/abhinav0107-collab/vaultpay/internal/service"
)

type DisputeHandler struct {
	disputeService *service.DisputeService
}

func NewDisputeHandler(s *service.DisputeService) *DisputeHandler {
	return &DisputeHandler{disputeService: s}
}

type DisputeRequest struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason"`
}

func (h *DisputeHandler) CreateDisputeHandler(w http.ResponseWriter, r *http.Request) {
	var req DisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Malformed dispute payload json structure", http.StatusBadRequest)
		return
	}

	// Securely generate a cryptographic dispute identifier
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	disputeID := fmt.Sprintf("dp_%x", b)

	dispute, err := h.disputeService.CreateDispute(disputeID, req.PaymentID, req.Reason)
	if err != nil {
		if err == service.ErrPaymentNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusConflict)
		}
		return
	}

	// Log out the webhook notification trace internally for observability
	fmt.Printf("📢 DISPUTE EVENT DISPATCHED: Payment %s is now locked under case %s\n", req.PaymentID, disputeID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(dispute)
}
