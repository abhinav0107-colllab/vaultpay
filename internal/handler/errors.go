package handler

import (
	"encoding/json"
	"net/http"
)

// ProblemDetails matches the RFC 7807 specification format
type ProblemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

// WriteProblemResponse compiles and transmits a uniform RFC 7807 error payload
func WriteProblemResponse(w http.ResponseWriter, instance string, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	// Fallback generic error URI types
	errType := "https://vaultpay.com/errors/internal-server-error"
	switch status {
	case http.StatusBadRequest:
		errType = "https://vaultpay.com/errors/bad-request"
	case http.StatusUnprocessableEntity:
		errType = "https://vaultpay.com/errors/unprocessable-entity"
	case http.StatusNotFound:
		errType = "https://vaultpay.com/errors/not-found"
	}

	problem := ProblemDetails{
		Type:     errType,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	}

	_ = json.NewEncoder(w).Encode(problem)
}
