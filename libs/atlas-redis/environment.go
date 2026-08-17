package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// EnvironmentRegistry provides environment-scoped CRUD operations for
// genuinely cross-tenant, control-plane state (FR-8.4). It mirrors
// TenantRegistry exactly, substituting env.Id for tenant.Model and
// environmentEntityKey for tenantEntityKey — see tenant_registry.go.
//
// UpdateWithTTL has no TenantRegistry counterpart; it is carried over from
// the bare Registry (registry.go) because atlas-data's ingestrun progress
// writer needs a TTL-preserving optimistic-lock update and this registry is
// its only environment-scoped equivalent.
type EnvironmentRegistry[K comparable, V any] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
	marshal   func(V) ([]byte, error)
	unmarshal func([]byte) (V, error)
}

func NewEnvironmentRegistry[K comparable, V any](client *goredis.Client, namespace string, keyFn func(K) string) *EnvironmentRegistry[K, V] {
	return &EnvironmentRegistry[K, V]{
		client:    client,
		namespace: namespace,
		keyFn:     keyFn,
		marshal:   func(v V) ([]byte, error) { return json.Marshal(v) },
		unmarshal: func(data []byte) (V, error) {
			var v V
			err := json.Unmarshal(data, &v)
			return v, err
		},
	}
}

func (r *EnvironmentRegistry[K, V]) entityKey(e env.Id, key K) string {
	return environmentEntityKey(r.namespace, e, r.keyFn(key))
}

func (r *EnvironmentRegistry[K, V]) Get(ctx context.Context, e env.Id, key K) (V, error) {
	rk := r.entityKey(e, key)
	data, err := r.client.Get(ctx, rk).Bytes()
	if errors.Is(err, goredis.Nil) {
		var zero V
		return zero, ErrNotFound
	}
	if err != nil {
		var zero V
		return zero, fmt.Errorf("redis get: %w", err)
	}
	return r.unmarshal(data)
}

// GetAllValues returns all values for an environment without requiring key reconstruction.
func (r *EnvironmentRegistry[K, V]) GetAllValues(ctx context.Context, e env.Id) ([]V, error) {
	var result []V
	pattern := environmentScanPattern(r.namespace, e)
	prefix := environmentEntityKey(r.namespace, e, "")
	var cursor uint64

	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan: %w", err)
		}

		if len(keys) > 0 {
			pipe := r.client.Pipeline()
			cmds := make([]*goredis.StringCmd, len(keys))
			for i, k := range keys {
				cmds[i] = pipe.Get(ctx, k)
			}
			_, _ = pipe.Exec(ctx)

			for i, cmd := range cmds {
				data, err := cmd.Bytes()
				if errors.Is(err, goredis.Nil) {
					continue
				}
				if err != nil {
					continue
				}
				// Skip internal keys.
				entityKeySuffix := strings.TrimPrefix(keys[i], prefix)
				if strings.HasPrefix(entityKeySuffix, "_") {
					continue
				}
				v, err := r.unmarshal(data)
				if err != nil {
					continue
				}
				result = append(result, v)
			}
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

func (r *EnvironmentRegistry[K, V]) Put(ctx context.Context, e env.Id, key K, value V) error {
	data, err := r.marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	rk := r.entityKey(e, key)
	return r.client.Set(ctx, rk, data, 0).Err()
}

func (r *EnvironmentRegistry[K, V]) PutWithTTL(ctx context.Context, e env.Id, key K, value V, ttl time.Duration) error {
	data, err := r.marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	rk := r.entityKey(e, key)
	return r.client.Set(ctx, rk, data, ttl).Err()
}

func (r *EnvironmentRegistry[K, V]) Remove(ctx context.Context, e env.Id, key K) error {
	rk := r.entityKey(e, key)
	return r.client.Del(ctx, rk).Err()
}

// RemoveExisting deletes the key and reports whether it existed. Redis DEL is
// atomic and returns the number of keys removed, so under concurrency exactly
// one caller observes true — the primitive callers need when a removal must
// also be an exclusive claim. Environment-scoped counterpart of
// TenantRegistry.RemoveExisting.
func (r *EnvironmentRegistry[K, V]) RemoveExisting(ctx context.Context, e env.Id, key K) (bool, error) {
	rk := r.entityKey(e, key)
	n, err := r.client.Del(ctx, rk).Result()
	if err != nil {
		return false, fmt.Errorf("redis del: %w", err)
	}
	return n == 1, nil
}

func (r *EnvironmentRegistry[K, V]) Update(ctx context.Context, e env.Id, key K, fn func(V) V) (V, error) {
	rk := r.entityKey(e, key)

	var result V
	txFn := func(tx *goredis.Tx) error {
		data, err := tx.Get(ctx, rk).Bytes()
		if errors.Is(err, goredis.Nil) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		current, err := r.unmarshal(data)
		if err != nil {
			return err
		}

		result = fn(current)
		newData, err := r.marshal(result)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, rk, newData, 0)
			return nil
		})
		return err
	}

	// Optimistic-lock retry on contention. goredis.TxFailedErr means another
	// writer modified the key between WATCH and EXEC; the safe response is to
	// re-read and re-apply fn. fn must be pure in its observable effects since
	// it may run multiple times.
	for i := 0; i < updateMaxRetries; i++ {
		err := r.client.Watch(ctx, txFn, rk)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, goredis.TxFailedErr) {
			continue
		}
		return result, err
	}
	return result, fmt.Errorf("optimistic lock failed after %d retries", updateMaxRetries)
}

// UpdateWithTTL is Update with a native Redis TTL re-applied on every write,
// so a key whose value must keep expiring across mutations (e.g. a
// bounded-lifetime progress record) does not go immortal on its first
// update. See Registry.UpdateWithTTL for the same rationale.
//
// fn may run multiple times (optimistic-lock retry) and must be pure.
func (r *EnvironmentRegistry[K, V]) UpdateWithTTL(ctx context.Context, e env.Id, key K, ttl time.Duration, fn func(V) V) (V, error) {
	rk := r.entityKey(e, key)

	var result V
	txFn := func(tx *goredis.Tx) error {
		data, err := tx.Get(ctx, rk).Bytes()
		if errors.Is(err, goredis.Nil) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		current, err := r.unmarshal(data)
		if err != nil {
			return err
		}

		result = fn(current)
		newData, err := r.marshal(result)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, rk, newData, ttl)
			return nil
		})
		return err
	}

	for i := 0; i < updateMaxRetries; i++ {
		err := r.client.Watch(ctx, txFn, rk)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, goredis.TxFailedErr) {
			continue
		}
		return result, err
	}
	return result, fmt.Errorf("optimistic lock failed after %d retries", updateMaxRetries)
}

func (r *EnvironmentRegistry[K, V]) Exists(ctx context.Context, e env.Id, key K) (bool, error) {
	rk := r.entityKey(e, key)
	n, err := r.client.Exists(ctx, rk).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}

// GetAllEntries returns a map of entity-key-suffix -> value for environment e.
// The suffix is the part of the Redis key after environmentEntityKey(namespace, e, "").
// Lets callers reconstruct typed keys without a second raw Scan. Skips "_"-prefixed internal keys.
func (r *EnvironmentRegistry[K, V]) GetAllEntries(ctx context.Context, e env.Id) (map[string]V, error) {
	result := make(map[string]V)
	pattern := environmentScanPattern(r.namespace, e)
	prefix := environmentEntityKey(r.namespace, e, "")
	var cursor uint64

	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan: %w", err)
		}

		if len(keys) > 0 {
			pipe := r.client.Pipeline()
			cmds := make([]*goredis.StringCmd, len(keys))
			for i, k := range keys {
				cmds[i] = pipe.Get(ctx, k)
			}
			_, _ = pipe.Exec(ctx)

			for i, cmd := range cmds {
				data, err := cmd.Bytes()
				if errors.Is(err, goredis.Nil) {
					continue
				}
				if err != nil {
					continue
				}
				// Skip internal keys.
				suffix := strings.TrimPrefix(keys[i], prefix)
				if strings.HasPrefix(suffix, "_") {
					continue
				}
				v, err := r.unmarshal(data)
				if err != nil {
					continue
				}
				result[suffix] = v
			}
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

// ClearByPrefix deletes every key for environment e whose entity-key-suffix begins with keyPrefix.
// Scan pattern = environmentEntityKey(r.namespace, e, keyPrefix) + "*". Pipelined DEL. Returns count.
func (r *EnvironmentRegistry[K, V]) ClearByPrefix(ctx context.Context, e env.Id, keyPrefix string) (int, error) {
	pattern := environmentEntityKey(r.namespace, e, keyPrefix) + "*"
	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()

	deleted := 0
	pipe := r.client.Pipeline()
	pipeSize := 0
	var firstErr error

	flushPipe := func() {
		if pipeSize == 0 {
			return
		}
		if _, err := pipe.Exec(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		pipe = r.client.Pipeline()
		pipeSize = 0
	}

	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
		deleted++
		pipeSize++
		if pipeSize >= 100 {
			flushPipe()
		}
	}
	flushPipe()

	if err := iter.Err(); err != nil && firstErr == nil {
		firstErr = err
	}
	return deleted, firstErr
}

// Client returns the underlying Redis client for advanced operations.
func (r *EnvironmentRegistry[K, V]) Client() *goredis.Client {
	return r.client
}

// Namespace returns the registry namespace.
func (r *EnvironmentRegistry[K, V]) Namespace() string {
	return r.namespace
}

// Clear deletes every entry for environment e in this registry's namespace.
// Implementation uses SCAN with COUNT=100 to enumerate keys matching
// environmentScanPattern(r.namespace, e), then pipelines DEL in batches of
// 100. Returns the number of keys deleted (0 if the namespace was already
// empty for this environment). On a partial failure mid-scan, returns
// (deleted_so_far, err) — the partial deletion is not rolled back; Redis
// converges on the next call.
func (r *EnvironmentRegistry[K, V]) Clear(ctx context.Context, e env.Id) (int, error) {
	pattern := environmentScanPattern(r.namespace, e)
	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()

	deleted := 0
	pipe := r.client.Pipeline()
	pipeSize := 0
	var firstErr error

	flushPipe := func() {
		if pipeSize == 0 {
			return
		}
		if _, err := pipe.Exec(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		pipe = r.client.Pipeline()
		pipeSize = 0
	}

	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
		deleted++
		pipeSize++
		if pipeSize >= 100 {
			flushPipe()
		}
	}
	flushPipe()

	if err := iter.Err(); err != nil && firstErr == nil {
		firstErr = err
	}
	return deleted, firstErr
}
