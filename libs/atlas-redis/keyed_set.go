package redis

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// TenantKeyedSet is a family of tenant-scoped SETs, one per key K.
// Key format: <prefix>:<namespace>:<tenantKey>:<keyFn(k)>.
type TenantKeyedSet[K comparable] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
}

func NewTenantKeyedSet[K comparable](client *goredis.Client, namespace string, keyFn func(K) string) *TenantKeyedSet[K] {
	return &TenantKeyedSet[K]{client: client, namespace: namespace, keyFn: keyFn}
}

func (s *TenantKeyedSet[K]) key(t tenant.Model, k K) string {
	return tenantEntityKey(s.namespace, t, s.keyFn(k))
}

func (s *TenantKeyedSet[K]) Add(ctx context.Context, t tenant.Model, k K, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SAdd(ctx, s.key(t, k), toIfaces(members)...).Err()
}

func (s *TenantKeyedSet[K]) Remove(ctx context.Context, t tenant.Model, k K, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SRem(ctx, s.key(t, k), toIfaces(members)...).Err()
}

func (s *TenantKeyedSet[K]) Members(ctx context.Context, t tenant.Model, k K) ([]string, error) {
	return s.client.SMembers(ctx, s.key(t, k)).Result()
}

// IsMember reports whether member is in the SET for (t, k).
func (s *TenantKeyedSet[K]) IsMember(ctx context.Context, t tenant.Model, k K, member string) (bool, error) {
	return s.client.SIsMember(ctx, s.key(t, k), member).Result()
}

// Clear removes the entire SET for (t, k).
func (s *TenantKeyedSet[K]) Clear(ctx context.Context, t tenant.Model, k K) error {
	return s.client.Del(ctx, s.key(t, k)).Err()
}
