package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockSubscriptionService perfectly mirrors the 5-parameter production method signature
type MockSubscriptionService struct {
	MockCreateFn func(id string, userID string, planID string, amount int64, period string) (interface{}, error)
}

func (m *MockSubscriptionService) CreateSubscription(id string, userID string, planID string, amount int64, period string) (interface{}, error) {
	return m.MockCreateFn(id, userID, planID, amount, period)
}

type testRequest struct {
	UserID        string `json:"user_id"`
	PlanID        string `json:"plan_id"`
	Amount        int64  `json:"amount"`
	BillingPeriod string `json:"billing_period"`
}

func TestCreateSubscriptionHandler_Success(t *testing.T) {
	mockService := &MockSubscriptionService{
		MockCreateFn: func(id, userID, planID string, amount int64, period string) (interface{}, error) {
			return map[string]string{"id": id, "status": "active"}, nil
		},
	}

	h := &SubscriptionHandler{subService: mockService}

	reqPayload := testRequest{
		UserID:        "user_456",
		PlanID:        "plan_premium",
		Amount:        2999,
		BillingPeriod: "monthly",
	}
	body, _ := json.Marshal(reqPayload)

	req, err := http.NewRequest("POST", "/v1/subscriptions", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.CreateSubscriptionHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code 201 Created, got %d", rr.Code)
	}
}

func TestCreateSubscriptionHandler_Failure_RFC7807(t *testing.T) {
	expectedErrText := "insufficient balance configurations to support active tier requirements"
	mockService := &MockSubscriptionService{
		MockCreateFn: func(id, userID, planID string, amount int64, period string) (interface{}, error) {
			// Trigger the error pathway
			return nil, errors.New(expectedErrText)
		},
	}

	h := &SubscriptionHandler{subService: mockService}

	reqPayload := testRequest{
		UserID:        "user_broke",
		PlanID:        "plan_premium",
		Amount:        99999,
		BillingPeriod: "yearly",
	}
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest("POST", "/v1/subscriptions", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.CreateSubscriptionHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status code 422 Unprocessable Entity, got %d", rr.Code)
	}

	var problem ProblemDetails
	if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
		t.Fatalf("Failed to decode error schema payload: %v", err)
	}

	if problem.Detail != expectedErrText {
		t.Errorf("Expected problem description field to reflect: %q, got: %q", expectedErrText, problem.Detail)
	}
}