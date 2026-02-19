package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// TenantRegistry provides tenant-scoped CRUD operations, replacing
// the map[tenant.Model]map[K]V + sync.RWMutex singleton pattern.
type TenantRegistry[K comparable, V any] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
	marshal   func(V) ([]byte, error)
	unmarshal func([]byte) (V, error)
}

func NewTenantRegistry[K comparable, V any](client *goredis.Client, namespace string, keyFn func(K) string) *TenantRegistry[K, V] {
	return &TenantRegistry[K, V]{
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

func (r *TenantRegistry[K, V]) entityKey(t tenant.Model, key K) string {
	return tenantEntityKey(r.namespace, t, r.keyFn(key))
}

func (r *TenantRegistry[K, V]) Get(ctx context.Context, t tenant.Model, key K) (V, error) {
	rk := r.entityKey(t, key)
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

// GetAllValues returns all values for a tenant without requiring key reconstruction.
func (r *TenantRegistry[K, V]) GetAllValues(ctx context.Context, t tenant.Model) ([]V, error) {
	var result []V
	pattern := tenantScanPattern(r.namespace, t)
	prefix := tenantEntityKey(r.namespace, t, "")
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

func (r *TenantRegistry[K, V]) Put(ctx context.Context, t tenant.Model, key K, value V) error {
	data, err := r.marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	rk := r.entityKey(t, key)
	return r.client.Set(ctx, rk, data, 0).Err()
}

func (r *TenantRegistry[K, V]) PutWithTTL(ctx context.Context, t tenant.Model, key K, value V, ttl time.Duration) error {
	data, err := r.marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	rk := r.entityKey(t, key)
	return r.client.Set(ctx, rk, data, ttl).Err()
}

func (r *TenantRegistry[K, V]) Remove(ctx context.Context, t tenant.Model, key K) error {
	rk := r.entityKey(t, key)
	return r.client.Del(ctx, rk).Err()
}

func (r *TenantRegistry[K, V]) Update(ctx context.Context, t tenant.Model, key K, fn func(V) V) (V, error) {
	rk := r.entityKey(t, key)

	var result V
	err := r.client.Watch(ctx, func(tx *goredis.Tx) error {
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
	}, rk)
	return result, err
}

func (r *TenantRegistry[K, V]) Exists(ctx context.Context, t tenant.Model, key K) (bool, error) {
	rk := r.entityKey(t, key)
	n, err := r.client.Exists(ctx, rk).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}

// Client returns the underlying Redis client for advanced operations.
func (r *TenantRegistry[K, V]) Client() *goredis.Client {
	return r.client
}

// Namespace returns the registry namespace.
func (r *TenantRegistry[K, V]) Namespace() string {
	return r.namespace
}
