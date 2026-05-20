package handler

import (
	"encoding/json"
	"net/http"

	"github.com/abhinav0107-collab/vaultpay/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(as *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: as}
}

type KeyRequest struct {
	UserID string `json:"user_id"`
	Label  string `json:"label"`
}

// CreateAPIKeyHandler decodes incoming web requests and responds with a fresh plain API key
func (h *AuthHandler) CreateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req KeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload syntax", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "Missing required payload field: user_id", http.StatusBadRequest)
		return
	}

	// Trigger our Day 3 security generator engine!
	plainKey, err := h.authService.GenerateAPIKey(req.UserID, req.Label)
	if err != nil {
		http.Error(w, "Internal system failure generating access key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"api_key": plainKey,
		"note":    "Copy this secret key down! It will never be shown in plaintext again.",
	})
}
