package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/service"
)

type JWTAuthHandler struct {
	jwtService *service.JWTAuthService
}

func NewJWTAuthHandler(s *service.JWTAuthService) *JWTAuthHandler {
	return &JWTAuthHandler{jwtService: s}
}

type TokenRequest struct {
	ClientID string `json:"client_id"`
	Secret   string `json:"client_secret"`
}

func (h *JWTAuthHandler) IssueTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Malformed token body request payload", http.StatusBadRequest)
		return
	}

	// Issue standard machine-to-machine service access tokens for integration
	token, err := h.jwtService.GenerateAccessToken(req.ClientID, "machine-to-machine")
	if err != nil {
		http.Error(w, "Failed to cryptographically sign token asset", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   "900", // 15 Minutes
	})
}

func (h *JWTAuthHandler) RevokeTokenHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")

	// Clean up any rogue double quotes or whitespace Swagger might have injected
	authHeader = strings.Trim(authHeader, "\" ")

	// Look for the "Bearer " prefix case-insensitively
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		http.Error(w, "Missing token target reference format", http.StatusBadRequest)
		return
	}

	// Safely extract the token string regardless of how many spaces Swagger added
	tokenStr := strings.TrimSpace(authHeader[7:])
	if tokenStr == "" {
		http.Error(w, "Empty token payload asset reference", http.StatusBadRequest)
		return
	}

	err := h.jwtService.RevokeToken(r.Context(), tokenStr, 15*time.Minute)
	if err != nil {
		http.Error(w, "Failed to register revocation signature state", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
