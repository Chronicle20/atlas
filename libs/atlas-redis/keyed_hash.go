package redis

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
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
