package redis

import (
	"context"
	"errors"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TenantKeyedHash is a family of tenant-scoped HASHes, one per key K.
// Key format: <prefix>:<namespace>:<tenantKey>:<keyFn(k)>.
type TenantKeyedHash[K comparable] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
}

func NewTenantKeyedHash[K comparable](client *goredis.Client, namespace string, keyFn func(K) string) *TenantKeyedHash[K] {
	return &TenantKeyedHash[K]{client: client, namespace: namespace, keyFn: keyFn}
}

func (h *TenantKeyedHash[K]) key(t tenant.Model, k K) string {
	return tenantEntityKey(h.namespace, t, h.keyFn(k))
}

func (h *TenantKeyedHash[K]) Set(ctx context.Context, t tenant.Model, k K, field, value string) error {
	return h.client.HSet(ctx, h.key(t, k), field, value).Err()
}

// SetNX sets field only if it does not yet exist; returns true if it was set.
func (h *TenantKeyedHash[K]) SetNX(ctx context.Context, t tenant.Model, k K, field, value string) (bool, error) {
	return h.client.HSetNX(ctx, h.key(t, k), field, value).Result()
}

func (h *TenantKeyedHash[K]) Get(ctx context.Context, t tenant.Model, k K, field string) (string, error) {
	v, err := h.client.HGet(ctx, h.key(t, k), field).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNotFound
	}
	return v, err
}

func (h *TenantKeyedHash[K]) Del(ctx context.Context, t tenant.Model, k K, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	return h.client.HDel(ctx, h.key(t, k), fields...).Err()
}

func (h *TenantKeyedHash[K]) Exists(ctx context.Context, t tenant.Model, k K, field string) (bool, error) {
	return h.client.HExists(ctx, h.key(t, k), field).Result()
}

func (h *TenantKeyedHash[K]) GetAll(ctx context.Context, t tenant.Model, k K) (map[string]string, error) {
	return h.client.HGetAll(ctx, h.key(t, k)).Result()
}

// DeleteKey removes the entire hash for (t, k).
func (h *TenantKeyedHash[K]) DeleteKey(ctx context.Context, t tenant.Model, k K) error {
	return h.client.Del(ctx, h.key(t, k)).Err()
}

// Key returns the fully-namespaced Redis key for (t, k). Exported for
// callers (e.g. atlas-maps) that run Lua scripts against the concrete key;
// key construction itself stays inside the lib.
func (h *TenantKeyedHash[K]) Key(t tenant.Model, k K) string { return h.key(t, k) }

// Len returns the number of fields in the hash for (t, k).
func (h *TenantKeyedHash[K]) Len(ctx context.Context, t tenant.Model, k K) (int64, error) {
	return h.client.HLen(ctx, h.key(t, k)).Result()
}

// ClearForTenantId deletes every hash whose key begins with
// <prefix>:<namespace>:<tenantId>. TenantKey(t) starts with the bare tenant
// UUID (see keys.go), so this pattern matches every (region, version) a
// given tenant ID has ever been keyed under without requiring the caller to
// know the tenant's current region/version — needed by TenantDeleted-style
// handlers that only carry the tenant ID. SCAN(COUNT=100) + pipelined DEL,
// mirroring KeyedHash.Clear. Returns the number of keys deleted.
func (h *TenantKeyedHash[K]) ClearForTenantId(ctx context.Context, tenantId uuid.UUID) (int, error) {
	pattern := namespacedKey(h.namespace, tenantId.String()) + keySeparator + "*"
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

// ClearAllAcrossTenants deletes every hash in the namespace, across every
// tenant. Deliberate, explicitly-named cross-tenant operation (D7) — for
// test-only full-namespace teardown, not for ordinary request paths.
func (h *TenantKeyedHash[K]) ClearAllAcrossTenants(ctx context.Context) (int, error) {
	pattern := namespacedKey(h.namespace) + keySeparator + "*"
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
