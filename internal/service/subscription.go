package service

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription record not found")
	ErrInvalidBillingPeriod = errors.New("billing period must be monthly or annual")
)

// Subscription represents a recurring user billing profile
type Subscription struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	PlanID             string    `json:"plan_id"`
	Amount             int64     `json:"amount"`         // Measured purely in Minor Integer Units
	BillingPeriod      string    `json:"billing_period"` // "monthly" or "annual"
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
}

type SubscriptionServiceImpl struct {
	// 🔥 DAY 22 TECH GAIN: Use sync.Map for high-concurrency, thread-safe memory state lookups
	store     sync.Map
	onceCheck sync.Once // Core pattern initialization gate
}

// Return the interface, but instantiate the concrete implementation struct
func NewSubscriptionService() SubscriptionService {
	return &SubscriptionServiceImpl{}
}

// 1. Updated interface to include BOTH methods
type SubscriptionService interface {
	// Change the first return type from *Subscription to interface{}
	CreateSubscription(id, userID, planID string, amount int64, period string) (interface{}, error)
	CalculateProration(subID string, newAmount int64) (int64, error)
}

// CreateSubscription maps out a brand new recurring transaction schedule
// 2. CHANGED RECEIVER TO: *SubscriptionServiceImpl AND RETURN TYPE TO: (interface{}, error)
func (s *SubscriptionServiceImpl) CreateSubscription(id, userID, planID string, amount int64, period string) (interface{}, error) {
	if period != "monthly" && period != "annual" {
		return nil, ErrInvalidBillingPeriod
	}

	startTime := time.Now()
	var endTime time.Time

	if period == "monthly" {
		endTime = startTime.AddDate(0, 1, 0) // Lookahead exactly 1 month
	} else {
		endTime = startTime.AddDate(1, 0, 0) // Lookahead exactly 1 year
	}

	sub := &Subscription{
		ID:                 id,
		UserID:             userID,
		PlanID:             planID,
		Amount:             amount,
		BillingPeriod:      period,
		CurrentPeriodStart: startTime,
		CurrentPeriodEnd:   endTime,
	}

	s.store.Store(id, sub)
	return sub, nil
}

// CalculateProration computes the remaining cash value credit of an existing tier
// and maps out the cost adjustments required to scale up to an alternative plan level.
// 3. CHANGED RECEIVER TO: *SubscriptionServiceImpl
func (s *SubscriptionServiceImpl) CalculateProration(subID string, newAmount int64) (int64, error) {
	value, exists := s.store.Load(subID)
	if !exists {
		return 0, ErrSubscriptionNotFound
	}

	sub := value.(*Subscription)

	now := time.Now()
	// ... leave the rest of your calculation code exactly as it is below this ...
	totalDuration := sub.CurrentPeriodEnd.Sub(sub.CurrentPeriodStart)
	remainingDuration := sub.CurrentPeriodEnd.Sub(now)

	if remainingDuration <= 0 {
		return newAmount, nil
	}

	// Calculate the credit left on the current plan using absolute integer precision ratios
	usedRatio := float64(now.Sub(sub.CurrentPeriodStart)) / float64(totalDuration)
	unusedRatio := 1.0 - usedRatio

	currentPlanUnusedCredit := float64(sub.Amount) * unusedRatio
	newPlanRemainingCost := float64(newAmount) * unusedRatio

	// The proration charge is the difference to pay for upgrading mid-cycle
	proratedCharge := newPlanRemainingCost - currentPlanUnusedCredit

	if proratedCharge < 0 {
		return 0, nil // Don't charge extra if they are downgrading; credit applies to the next invoice cycle
	}

	return int64(proratedCharge), nil
}
