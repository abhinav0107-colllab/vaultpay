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

func TestCreateChargeHandler_ComprehensiveValidation(t *testing.T) {
	// 1. Define our test cases table grid matrix
	testCases := []struct {
		name           string
		requestPayload string
		expectedStatus int
	}{
		{
			name:           "Failure Path - Missing User ID String",
			requestPayload: `{"user_id": "", "amount": 25000, "currency": "INR"}`,
			expectedStatus: http.StatusUnprocessableEntity, // 422
		},
		{
			name:           "Failure Path - Negative Or Zero Transaction Amount",
			requestPayload: `{"user_id": "user_uuid_999", "amount": -500, "currency": "INR"}`,
			expectedStatus: http.StatusUnprocessableEntity, // 422
		},
		{
			name:           "Failure Path - Malformed Broken JSON Body",
			requestPayload: `{"user_id": "user_uuid_111", "amount": 5000,`, // missing bracket
			expectedStatus: http.StatusBadRequest,                          // 400
		},
	}

	// 2. Loop through our matrix rows executing isolated testing conditions
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup clean infrastructure dependencies for every test step iteration
			paymentRepo := repository.NewPaymentRepository(nil)
			// Update line 43 to include the nil outbox parameter
			paymentServ := service.NewPaymentService(paymentRepo, nil) // Update line 44 to include the nil lock parameter
			// Add an asterisk in front of paymentServ to pass it matching the interface signature
			paymentHand := handler.NewPaymentHandler(*paymentServ, nil)
			// Construct virtual HTTP request pipeline inside memory architecture
			req, err := http.NewRequest("POST", "/v1/charges", bytes.NewBuffer([]byte(tc.requestPayload)))
			if err != nil {
				t.Fatalf("Failed to initialize test request entity context: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Instantiate our response stream recorder state tracker
			rr := httptest.NewRecorder()
			handlerFunc := http.HandlerFunc(paymentHand.CreateChargeHandler)

			// Fire request directly into handler context
			handlerFunc.ServeHTTP(rr, req)

			// Verify status response matching our expected limits matrices
			if rr.Code != tc.expectedStatus {
				t.Errorf("[%s] Unexpected status code returned: got %d, wanted %d", tc.name, rr.Code, tc.expectedStatus)
			}
		})
	}
}
