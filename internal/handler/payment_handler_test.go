package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abhinav0107-collab/vaultpay/internal/handler"
	"github.com/abhinav0107-collab/vaultpay/internal/repository"
	"github.com/abhinav0107-collab/vaultpay/internal/service"
)

// TestCreateChargeHandler_Validation checks that our HTTP pipeline flags empty parameters cleanly
func TestCreateChargeHandler_Validation(t *testing.T) {
	// 1. Setup a clean, independent testing stack instance (passing nil for database since we expect validation to fail early)
	paymentRepo := repository.NewPaymentRepository(nil)
	paymentServ := service.NewPaymentService(paymentRepo)
	paymentHand := handler.NewPaymentHandler(paymentServ)

	// 2. Draft a malformed JSON payload missing a valid target User ID string
	invalidJSON := []byte(`{"user_id": "", "amount": 25000, "currency": "INR"}`)

	// 3. Construct a mock HTTP request inside computer memory
	req, err := http.NewRequest("POST", "/v1/charges", bytes.NewBuffer(invalidJSON))
	if err != nil {
		t.Fatalf("Failed to initialize test request entity context: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 4. Instantiate a virtual response recorder to capture the handler's output signatures
	rr := httptest.NewRecorder()

	// 5. Fire the request directly into the isolated Go handler function
	handlerFunc := http.HandlerFunc(paymentHand.CreateChargeHandler)
	handlerFunc.ServeHTTP(rr, req)

	// 6. Assertions: Verify the system responded with a 422 Unprocessable Entity code
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Unexpected status code returned: got %d, wanted %d", rr.Code, http.StatusUnprocessableEntity)
	}
}
