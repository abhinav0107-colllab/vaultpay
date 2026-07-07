package database

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrLockHeld is returned when a parallel thread already occupies the transaction lock
var ErrLockHeld = errors.New("concurrent transaction execution blocked: lock is already held")

type DistributedLock struct {
	client *redis.Client
}

// NewDistributedLock instantiates our app-level mutual exclusion manager
func NewDistributedLock(client *redis.Client) *DistributedLock {
	return &DistributedLock{client: client}
}

// AcquireLock attempts to claim a unique atomic key in Redis with a TTL expiration
func (l *DistributedLock) AcquireLock(ctx context.Context, key string, expiration time.Duration) (string, error) {
	// Generate a high-precision unique token so only this specific thread can release the lock
	token := time.Now().Format(time.RFC3339Nano)

	// SET lock:payment:key token NX PX expiration
	succeeded, err := l.client.SetNX(ctx, "lock:payment:"+key, token, expiration).Result()
	if err != nil {
		return "", err
	}

	if !succeeded {
		return "", ErrLockHeld
	}

	return token, nil
}

// ReleaseLock removes the key safely using an atomic Lua script to prevent cross-thread deletion
func (l *DistributedLock) ReleaseLock(ctx context.Context, key string, token string) error {
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := l.client.Eval(ctx, luaScript, []string{"lock:payment:" + key}, token).Result()
	return err
}
