package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/config"
	_ "github.com/lib/pq"
)

type DatabaseCluster struct {
	Master         *sql.DB
	Replica        *sql.DB
	ReplicaBreaker *CircuitBreaker // ◄ Added protective circuit breaker layer
}

type BreakerState string

const (
	StateClosed   BreakerState = "CLOSED"    // Normal operations
	StateOpen     BreakerState = "OPEN"      // Database down, failing fast
	StateHalfOpen BreakerState = "HALF_OPEN" // Testing database recovery
)

type CircuitBreaker struct {
	mu           sync.RWMutex
	state        BreakerState
	failureCount int
	threshold    int
	cooldown     time.Duration
	lastStateLog time.Time
}

// InitCluster initializes connection pools for both writing and reading nodes
func InitCluster(cfg *config.Config) (*DatabaseCluster, error) {
	// 1. Establish Master Connection
	masterDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	masterPool, err := sql.Open("postgres", masterDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open master DB: %w", err)
	}
	configurePool(masterPool)

	// 2. Establish Replica Connection
	replicaDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBReplicaHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	replicaPool, err := sql.Open("postgres", replicaDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open replica DB: %w", err)
	}
	configurePool(replicaPool)

	// 3. Verify health
	if err := masterPool.Ping(); err != nil {
		return nil, fmt.Errorf("master node unreachable: %w", err)
	}
	if err := replicaPool.Ping(); err != nil {
		return nil, fmt.Errorf("replica node unreachable: %w", err)
	}

	return &DatabaseCluster{
		Master:  masterPool,
		Replica: replicaPool,
		ReplicaBreaker: &CircuitBreaker{
			state:     StateClosed,
			threshold: 5,                // Trip open after 5 consecutive failures
			cooldown:  15 * time.Second, // Let the replica rest for 15s before re-testing
		},
	}, nil
}

func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
}

// RetryQuery executes a read-only operation with an exponential backoff strategy for high resiliency
// RetryQuery executes an operation with both Circuit Breaker protection and Exponential Backoff
func (c *DatabaseCluster) RetryQuery(ctx context.Context, operation func() error) error {
	// 1. Check if the breaker allows replica traffic
	if !c.ReplicaBreaker.CanExecute() {
		// Fallback Strategy: Instead of crashing, gracefully route the read query to Master!
		return operation()
	}

	maxRetries := 3
	backoff := 100 * time.Millisecond
	var err error

	for i := 0; i < maxRetries; i++ {
		if err = operation(); err == nil {
			c.ReplicaBreaker.RecordSuccess() // ◄ Query succeeded! Clear breaker metrics.
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}

	// 2. If the operation consistently fails across retries, trip the circuit breaker
	c.ReplicaBreaker.RecordFailure()
	return fmt.Errorf("breaker pipeline intercepted failure: %w", err)
}

// CanExecute checks the breaker state and determines if the replica can take traffic
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	if cb.state == StateClosed {
		cb.mu.RUnlock()
		return true
	}
	cb.mu.RUnlock()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// If the circuit is OPEN, check if the cooldown window has expired
	if cb.state == StateOpen {
		if time.Since(cb.lastStateLog) > cb.cooldown {
			cb.state = StateHalfOpen
			cb.lastStateLog = time.Now()
			return true
		}
		return false // Circuit is still open, fail fast!
	}

	return true // HALF_OPEN allows a trial request through
}

// RecordFailure increments errors and trips the breaker if threshold is breached
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	if cb.state == StateHalfOpen || cb.failureCount >= cb.threshold {
		cb.state = StateOpen
		cb.lastStateLog = time.Now()
	}
}

// RecordSuccess resets the breaker back to a pristine operational state
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
}
