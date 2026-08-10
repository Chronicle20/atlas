package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// decrByIfExistsScript atomically decrements an EXISTING counter and
// refreshes its TTL, returning {newValue, 1}. A missing key is NOT created
// (a bare DECRBY would create it at -delta, turning "state lost" into an
// instant zero-crossing); it returns {0, 0} so the caller can take a lazy
// re-initialization path instead.
var decrByIfExistsScript = goredis.NewScript(`
if redis.call("exists", KEYS[1]) == 1 then
	local v = redis.call("decrby", KEYS[1], ARGV[1])
	redis.call("pexpire", KEYS[1], ARGV[2])
	return {v, 1}
else
	return {0, 0}
end`)

// initIfMissingAndDecrByScript atomically initializes an absent counter to
// ARGV[1] (leaving an existing value untouched) and then decrements by
// ARGV[2], refreshing the TTL. Both the presence check and the decrement
// happen inside one Redis-serialized script execution, so two concurrent
// callers racing a missing key can never both seed the same baseline and
// independently subtract from it (which would lose one decrement) or both
// observe the same post-seed crossing (which would double-break). The
// second (and every later) concurrent caller instead decrements the value
// the first caller already seeded.
var initIfMissingAndDecrByScript = goredis.NewScript(`
if redis.call("exists", KEYS[1]) == 0 then
	redis.call("set", KEYS[1], ARGV[1])
end
local v = redis.call("decrby", KEYS[1], ARGV[2])
redis.call("pexpire", KEYS[1], ARGV[3])
return v`)

// incrWithTTLScript increments a counter and sets its TTL only when the key is
// being created. A plain INCR leaves a new key with no expiry, which makes a
// rate-limit window permanent; refreshing the TTL on EVERY increment instead
// turns a fixed window into a sliding one an attacker can keep alive forever.
// Setting it exactly once, at creation, is the fixed-window semantics callers
// expect.
var incrWithTTLScript = goredis.NewScript(`
local v = redis.call("incr", KEYS[1])
if v == 1 then
	redis.call("pexpire", KEYS[1], ARGV[1])
end
return v`)

// TenantCounter is a tenant-scoped int64 counter with a TTL-bounded
// lifetime. Decrements are serialized by Redis, so concurrent callers never
// lose an update and exactly one caller observes any given zero crossing
// (newValue <= 0 && newValue+delta > 0).
type TenantCounter struct {
	client    *goredis.Client
	namespace string
}

func NewTenantCounter(client *goredis.Client, namespace string) *TenantCounter {
	return &TenantCounter{client: client, namespace: namespace}
}

func (c *TenantCounter) entityKey(t tenant.Model, key string) string {
	return tenantEntityKey(c.namespace, t, key)
}

// Set stores value with ttl, replacing any prior value and TTL.
func (c *TenantCounter) Set(ctx context.Context, t tenant.Model, key string, value int64, ttl time.Duration) error {
	if err := c.client.Set(ctx, c.entityKey(t, key), value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// DecrByIfExists atomically decrements the counter by delta and refreshes
// its TTL. Returns existed=false (without creating the key) when the
// counter is absent.
func (c *TenantCounter) DecrByIfExists(ctx context.Context, t tenant.Model, key string, delta int64, ttl time.Duration) (int64, bool, error) {
	res, err := decrByIfExistsScript.Run(ctx, c.client, []string{c.entityKey(t, key)}, delta, ttl.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, false, fmt.Errorf("redis decr-if-exists: %w", err)
	}
	if len(res) != 2 {
		return 0, false, fmt.Errorf("redis decr-if-exists: unexpected reply length %d", len(res))
	}
	return res[0], res[1] == 1, nil
}

// InitIfMissingAndDecrBy atomically seeds the counter to initial if (and
// only if) it is currently absent, then decrements it by delta and refreshes
// its TTL, returning the resulting value. Use this instead of a plain
// Set-then-decrement pair when the caller cannot tell in advance whether the
// counter still exists (e.g. lazy re-initialization after a TTL expiry or a
// Redis restart) — a Set-then-decrement race across concurrent callers can
// lose a decrement or double-count a zero crossing; this cannot.
func (c *TenantCounter) InitIfMissingAndDecrBy(ctx context.Context, t tenant.Model, key string, initial int64, delta int64, ttl time.Duration) (int64, error) {
	v, err := initIfMissingAndDecrByScript.Run(ctx, c.client, []string{c.entityKey(t, key)}, initial, delta, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, fmt.Errorf("redis init-if-missing-and-decr: %w", err)
	}
	return v, nil
}

// IncrWithTTL increments the counter by one and returns the new value. The TTL
// is applied only when the counter is created, giving a fixed window that
// starts at the first increment.
func (c *TenantCounter) IncrWithTTL(ctx context.Context, t tenant.Model, key string, ttl time.Duration) (int64, error) {
	v, err := incrWithTTLScript.Run(ctx, c.client, []string{c.entityKey(t, key)}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, fmt.Errorf("redis incr-with-ttl: %w", err)
	}
	return v, nil
}

// Get returns the counter's current value, or 0 when the key is absent. It
// never creates or mutates the key.
func (c *TenantCounter) Get(ctx context.Context, t tenant.Model, key string) (int64, error) {
	v, err := c.client.Get(ctx, c.entityKey(t, key)).Int64()
	if errors.Is(err, goredis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("redis get counter: %w", err)
	}
	return v, nil
}

// Remove deletes the counter. Removing a missing key is a no-op.
func (c *TenantCounter) Remove(ctx context.Context, t tenant.Model, key string) error {
	if err := c.client.Del(ctx, c.entityKey(t, key)).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
