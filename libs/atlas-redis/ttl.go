package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// TTLRegistry extends TenantRegistry with TTL-aware operations.
// It uses a sorted set to track expiration timestamps for efficient
// expired-entry retrieval (popExpired pattern).
type TTLRegistry[K comparable, V any] struct {
	*TenantRegistry[K, V]
	defaultTTL time.Duration
	nowFn      func() time.Time
}

func NewTTLRegistry[K comparable, V any](client *goredis.Client, namespace string, keyFn func(K) string, defaultTTL time.Duration) *TTLRegistry[K, V] {
	return &TTLRegistry[K, V]{
		TenantRegistry: NewTenantRegistry[K, V](client, namespace, keyFn),
		defaultTTL:     defaultTTL,
		nowFn:          time.Now,
	}
}

// SetNowFunc overrides the clock function, primarily for testing.
func (r *TTLRegistry[K, V]) SetNowFunc(fn func() time.Time) {
	r.nowFn = fn
}

func (r *TTLRegistry[K, V]) expirySetKey(t tenant.Model) string {
	return tenantEntityKey(r.namespace, t, "_expiry")
}

// Put stores a value with the default TTL and tracks it in the expiry set.
func (r *TTLRegistry[K, V]) Put(ctx context.Context, t tenant.Model, key K, value V) error {
	return r.PutWithTTL(ctx, t, key, value, r.defaultTTL)
}

// PutWithTTL stores a value with a custom TTL and tracks it in the expiry set.
func (r *TTLRegistry[K, V]) PutWithTTL(ctx context.Context, t tenant.Model, key K, value V, ttl time.Duration) error {
	data, err := r.marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	rk := r.entityKey(t, key)
	expiryKey := r.expirySetKey(t)
	expireAt := float64(r.nowFn().Add(ttl).UnixMilli())

	pipe := r.client.Pipeline()
	// Store data without Redis native TTL — PopExpired needs to read it.
	// The sorted set is the source of truth for expiration.
	pipe.Set(ctx, rk, data, 0)
	pipe.ZAdd(ctx, expiryKey, goredis.Z{Score: expireAt, Member: rk})
	_, err = pipe.Exec(ctx)
	return err
}

// PopExpired returns and removes all expired entries across all tenants.
// This matches the existing popExpired pattern used by expressions, cashshop, etc.
func (r *TTLRegistry[K, V]) PopExpired(ctx context.Context, t tenant.Model) ([]V, error) {
	expiryKey := r.expirySetKey(t)
	now := float64(r.nowFn().UnixMilli())

	// Get all members with expiry <= now.
	members, err := r.client.ZRangeByScore(ctx, expiryKey, &goredis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("redis zrangebyscore: %w", err)
	}

	if len(members) == 0 {
		return nil, nil
	}

	var results []V

	// Fetch values and remove in a pipeline.
	pipe := r.client.Pipeline()
	getCmds := make([]*goredis.StringCmd, len(members))
	for i, member := range members {
		getCmds[i] = pipe.Get(ctx, member)
	}
	_, err = pipe.Exec(ctx)
	if err != nil && !errors.Is(err, goredis.Nil) {
		return nil, fmt.Errorf("redis pipeline get expired: %w", err)
	}

	// Collect results.
	keysToRemove := make([]string, 0, len(members))
	for i, cmd := range getCmds {
		data, err := cmd.Bytes()
		if errors.Is(err, goredis.Nil) {
			// Key already expired via TTL, just clean up the sorted set.
			keysToRemove = append(keysToRemove, members[i])
			continue
		}
		if err != nil {
			continue
		}
		v, err := r.unmarshal(data)
		if err != nil {
			continue
		}
		results = append(results, v)
		keysToRemove = append(keysToRemove, members[i])
	}

	// Clean up: remove from sorted set and delete keys.
	if len(keysToRemove) > 0 {
		pipe = r.client.Pipeline()
		membersToRemove := make([]any, len(keysToRemove))
		for i, k := range keysToRemove {
			membersToRemove[i] = k
		}
		pipe.ZRem(ctx, expiryKey, membersToRemove...)
		pipe.Del(ctx, keysToRemove...)
		_, _ = pipe.Exec(ctx)
	}

	return results, nil
}

// PopExpiredWithKeys returns expired entries with their serialized keys for callers that need them.
func (r *TTLRegistry[K, V]) PopExpiredWithKeys(ctx context.Context, t tenant.Model) (map[string]V, error) {
	expiryKey := r.expirySetKey(t)
	now := float64(r.nowFn().UnixMilli())

	members, err := r.client.ZRangeByScore(ctx, expiryKey, &goredis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("redis zrangebyscore: %w", err)
	}

	if len(members) == 0 {
		return nil, nil
	}

	results := make(map[string]V)

	pipe := r.client.Pipeline()
	getCmds := make([]*goredis.StringCmd, len(members))
	for i, member := range members {
		getCmds[i] = pipe.Get(ctx, member)
	}
	_, err = pipe.Exec(ctx)
	if err != nil && !errors.Is(err, goredis.Nil) {
		return nil, fmt.Errorf("redis pipeline get expired: %w", err)
	}

	keysToRemove := make([]string, 0, len(members))
	for i, cmd := range getCmds {
		data, err := cmd.Bytes()
		if errors.Is(err, goredis.Nil) {
			keysToRemove = append(keysToRemove, members[i])
			continue
		}
		if err != nil {
			continue
		}
		v, err := r.unmarshal(data)
		if err != nil {
			continue
		}
		results[members[i]] = v
		keysToRemove = append(keysToRemove, members[i])
	}

	if len(keysToRemove) > 0 {
		pipe = r.client.Pipeline()
		membersToRemove := make([]any, len(keysToRemove))
		for i, k := range keysToRemove {
			membersToRemove[i] = k
		}
		pipe.ZRem(ctx, expiryKey, membersToRemove...)
		pipe.Del(ctx, keysToRemove...)
		_, _ = pipe.Exec(ctx)
	}

	return results, nil
}

// Remove removes an entry and its expiry tracking.
func (r *TTLRegistry[K, V]) Remove(ctx context.Context, t tenant.Model, key K) error {
	rk := r.entityKey(t, key)
	expiryKey := r.expirySetKey(t)

	pipe := r.client.Pipeline()
	pipe.Del(ctx, rk)
	pipe.ZRem(ctx, expiryKey, rk)
	_, err := pipe.Exec(ctx)
	return err
}

