package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const defaultLockTTL = 30 * time.Second

// Lock provides distributed locking via Redis SET NX EX.
type Lock struct {
	client    *goredis.Client
	namespace string
	ttl       time.Duration
}

func NewLock(client *goredis.Client, namespace string) *Lock {
	return &Lock{
		client:    client,
		namespace: namespace,
		ttl:       defaultLockTTL,
	}
}

func NewLockWithTTL(client *goredis.Client, namespace string, ttl time.Duration) *Lock {
	return &Lock{
		client:    client,
		namespace: namespace,
		ttl:       ttl,
	}
}

func (l *Lock) lockKey(key string) string {
	return namespacedKey(l.namespace, "_lock", key)
}

// Acquire attempts to acquire a distributed lock. Returns true if the lock was acquired.
func (l *Lock) Acquire(ctx context.Context, key string) (bool, error) {
	rk := l.lockKey(key)
	ok, err := l.client.SetNX(ctx, rk, "1", l.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}
	return ok, nil
}

// AcquireWithTTL attempts to acquire a lock with a custom TTL.
func (l *Lock) AcquireWithTTL(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	rk := l.lockKey(key)
	ok, err := l.client.SetNX(ctx, rk, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}
	return ok, nil
}

// Release releases a distributed lock.
func (l *Lock) Release(ctx context.Context, key string) error {
	rk := l.lockKey(key)
	return l.client.Del(ctx, rk).Err()
}

// Extend resets the TTL on an existing lock.
func (l *Lock) Extend(ctx context.Context, key string) (bool, error) {
	rk := l.lockKey(key)
	return l.client.Expire(ctx, rk, l.ttl).Result()
}
