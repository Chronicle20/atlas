package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("not found")

// Registry is the core generic type replacing map-based registries.
type Registry[K comparable, V any] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
	marshal   func(V) ([]byte, error)
	unmarshal func([]byte) (V, error)
}

func NewRegistry[K comparable, V any](client *goredis.Client, namespace string, keyFn func(K) string) *Registry[K, V] {
	return &Registry[K, V]{
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

func (r *Registry[K, V]) Get(ctx context.Context, key K) (V, error) {
	rk := namespacedKey(r.namespace, r.keyFn(key))
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

func (r *Registry[K, V]) Put(ctx context.Context, key K, value V) error {
	data, err := r.marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	rk := namespacedKey(r.namespace, r.keyFn(key))
	return r.client.Set(ctx, rk, data, 0).Err()
}

func (r *Registry[K, V]) Remove(ctx context.Context, key K) error {
	rk := namespacedKey(r.namespace, r.keyFn(key))
	return r.client.Del(ctx, rk).Err()
}

// updateMaxRetries bounds the optimistic-lock retry loop in Update. Set high
// enough to absorb contention from N concurrent writers hammering the same key
// (each lost race only burns one retry; expected per-op retries scale roughly
// with the number of contenders). Migrating monster-store callers off a single
// server-side Lua script (atomic by construction) onto optimistic Watch raises
// the contention budget needed to maintain "no silently dropped writes" — which
// matters for live combat state.
const updateMaxRetries = 1000

func (r *Registry[K, V]) Update(ctx context.Context, key K, fn func(V) V) (V, error) {
	rk := namespacedKey(r.namespace, r.keyFn(key))

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

// PutWithTTL stores value under key with a native Redis TTL.
func (r *Registry[K, V]) PutWithTTL(ctx context.Context, key K, value V, ttl time.Duration) error {
	data, err := r.marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	rk := namespacedKey(r.namespace, r.keyFn(key))
	return r.client.Set(ctx, rk, data, ttl).Err()
}

// GetAll returns every value in this env-global namespace (SCAN + pipelined GET).
// Skips internal keys whose suffix begins with "_". Mirrors TenantRegistry.GetAllValues.
func (r *Registry[K, V]) GetAll(ctx context.Context) ([]V, error) {
	var result []V
	pattern := namespacedKey(r.namespace, "*")
	prefix := namespacedKey(r.namespace, "")
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

// Clear deletes every key in this namespace (SCAN COUNT=100 + pipelined DEL).
// Returns count deleted. Mirrors TenantRegistry.Clear.
func (r *Registry[K, V]) Clear(ctx context.Context) (int, error) {
	pattern := namespacedKey(r.namespace, "*")
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

// ClearByPrefix deletes every key in this namespace whose entity-key begins
// with keyPrefix. Scan pattern = namespacedKey(namespace, keyPrefix) + "*"
// (no extra separator). Pipelined DEL. Returns count deleted.
func (r *Registry[K, V]) ClearByPrefix(ctx context.Context, keyPrefix string) (int, error) {
	pattern := namespacedKey(r.namespace, keyPrefix) + "*"
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

func (r *Registry[K, V]) Client() *goredis.Client {
	return r.client
}

func (r *Registry[K, V]) Namespace() string {
	return r.namespace
}

func (r *Registry[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	rk := namespacedKey(r.namespace, r.keyFn(key))
	n, err := r.client.Exists(ctx, rk).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}
