package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const defaultLockTTL = 30 * time.Second

// releaseTokenScript is an atomic compare-and-delete: deletes the key only when
// its current value equals the caller's token, returning 1 if deleted, 0 otherwise.
var releaseTokenScript = goredis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end`)

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

// AcquireWithToken attempts SET NX on the lock key with the caller's token as the
// value and a custom TTL. Returns true if acquired. The token lets the holder
// release only its own lock via ReleaseToken.
func (l *Lock) AcquireWithToken(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	rk := l.lockKey(key)
	ok, err := l.client.SetNX(ctx, rk, token, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}
	return ok, nil
}

// ForceAcquire unconditionally takes the lock for the caller's token with a TTL
// (used as a timeout fallback when a stale holder won't release). Overwrites any
// existing holder.
func (l *Lock) ForceAcquire(ctx context.Context, key, token string, ttl time.Duration) error {
	rk := l.lockKey(key)
	return l.client.Set(ctx, rk, token, ttl).Err()
}

// ReleaseToken releases the lock ONLY if its current value equals token
// (compare-and-delete, atomic via Lua). Returns true if this caller's lock was
// released, false if the lock was held by someone else or already gone.
func (l *Lock) ReleaseToken(ctx context.Context, key, token string) (bool, error) {
	rk := l.lockKey(key)
	res, err := releaseTokenScript.Run(ctx, l.client, []string{rk}, token).Int()
	if err != nil {
		return false, fmt.Errorf("redis release-token: %w", err)
	}
	return res == 1, nil
}
