package redis

import (
	"context"
	"errors"

	goredis "github.com/redis/go-redis/v9"
)

// Hash is an env-global Redis HASH whose key is namespaced via KeyPrefix().
type Hash struct {
	client    *goredis.Client
	namespace string
}

func NewHash(client *goredis.Client, namespace string) *Hash {
	return &Hash{client: client, namespace: namespace}
}

func (h *Hash) key() string { return namespacedKey(h.namespace) }

func (h *Hash) Set(ctx context.Context, field, value string) error {
	return h.client.HSet(ctx, h.key(), field, value).Err()
}

func (h *Hash) Get(ctx context.Context, field string) (string, error) {
	v, err := h.client.HGet(ctx, h.key(), field).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNotFound
	}
	return v, err
}

func (h *Hash) Del(ctx context.Context, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	return h.client.HDel(ctx, h.key(), fields...).Err()
}

func (h *Hash) Exists(ctx context.Context, field string) (bool, error) {
	return h.client.HExists(ctx, h.key(), field).Result()
}

func (h *Hash) GetAll(ctx context.Context) (map[string]string, error) {
	return h.client.HGetAll(ctx, h.key()).Result()
}

// KeyedHash is a family of env-global HASHes, one per key K. The Lua-script
// callers (atlas-maps) obtain the concrete Redis key via Key(k) and run their
// scripts against it; Key construction stays inside the lib.
type KeyedHash[K comparable] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
}

func NewKeyedHash[K comparable](client *goredis.Client, namespace string, keyFn func(K) string) *KeyedHash[K] {
	return &KeyedHash[K]{client: client, namespace: namespace, keyFn: keyFn}
}

// Key returns the fully-namespaced Redis key for k.
func (h *KeyedHash[K]) Key(k K) string { return namespacedKey(h.namespace, h.keyFn(k)) }

func (h *KeyedHash[K]) Set(ctx context.Context, k K, field, value string) error {
	return h.client.HSet(ctx, h.Key(k), field, value).Err()
}

func (h *KeyedHash[K]) Get(ctx context.Context, k K, field string) (string, error) {
	v, err := h.client.HGet(ctx, h.Key(k), field).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNotFound
	}
	return v, err
}

func (h *KeyedHash[K]) Del(ctx context.Context, k K, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	return h.client.HDel(ctx, h.Key(k), fields...).Err()
}

func (h *KeyedHash[K]) Exists(ctx context.Context, k K, field string) (bool, error) {
	return h.client.HExists(ctx, h.Key(k), field).Result()
}

func (h *KeyedHash[K]) GetAll(ctx context.Context, k K) (map[string]string, error) {
	return h.client.HGetAll(ctx, h.Key(k)).Result()
}

func (h *KeyedHash[K]) Len(ctx context.Context, k K) (int64, error) {
	return h.client.HLen(ctx, h.Key(k)).Result()
}

// DeleteKey removes the entire hash for k.
func (h *KeyedHash[K]) DeleteKey(ctx context.Context, k K) error {
	return h.client.Del(ctx, h.Key(k)).Err()
}

// Clear deletes every hash whose key begins with
// namespacedKey(namespace, segments...). With no segments it clears the whole
// namespace. SCAN(COUNT=100) + pipelined DEL, mirroring TenantRegistry.Clear.
// Returns the number of keys deleted.
func (h *KeyedHash[K]) Clear(ctx context.Context, segments ...string) (int, error) {
	var pattern string
	if len(segments) == 0 {
		pattern = namespacedKey(h.namespace) + keySeparator + "*"
	} else {
		pattern = namespacedKey(h.namespace, segments...) + keySeparator + "*"
	}
	iter := h.client.Scan(ctx, 0, pattern, 100).Iterator()
	deleted := 0
	pipe := h.client.Pipeline()
	pipeSize := 0
	var firstErr error
	flush := func() {
		if pipeSize == 0 {
			return
		}
		if _, err := pipe.Exec(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		pipe = h.client.Pipeline()
		pipeSize = 0
	}
	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
		deleted++
		pipeSize++
		if pipeSize >= 100 {
			flush()
		}
	}
	flush()
	if err := iter.Err(); err != nil && firstErr == nil {
		firstErr = err
	}
	return deleted, firstErr
}
