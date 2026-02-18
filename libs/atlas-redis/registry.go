package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

func (r *Registry[K, V]) Update(ctx context.Context, key K, fn func(V) V) (V, error) {
	rk := namespacedKey(r.namespace, r.keyFn(key))

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
