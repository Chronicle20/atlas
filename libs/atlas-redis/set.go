// libs/atlas-redis/set.go
package redis

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

func toIfaces(members []string) []interface{} {
	out := make([]interface{}, len(members))
	for i, m := range members {
		out[i] = m
	}
	return out
}

// Set is an env-global Redis SET whose key is namespaced via KeyPrefix().
// Use for cross-tenant-within-env aggregate sets (e.g. "drops:all").
type Set struct {
	client    *goredis.Client
	namespace string
}

func NewSet(client *goredis.Client, namespace string) *Set {
	return &Set{client: client, namespace: namespace}
}

func (s *Set) key() string { return namespacedKey(s.namespace) }

func (s *Set) Add(ctx context.Context, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SAdd(ctx, s.key(), toIfaces(members)...).Err()
}

func (s *Set) Remove(ctx context.Context, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SRem(ctx, s.key(), toIfaces(members)...).Err()
}

func (s *Set) Members(ctx context.Context) ([]string, error) {
	return s.client.SMembers(ctx, s.key()).Result()
}

func (s *Set) IsMember(ctx context.Context, member string) (bool, error) {
	return s.client.SIsMember(ctx, s.key(), member).Result()
}

func (s *Set) Size(ctx context.Context) (int64, error) {
	return s.client.SCard(ctx, s.key()).Result()
}

// TenantSet is a tenant-scoped Redis SET: one SET per tenant under namespace.
type TenantSet struct {
	client    *goredis.Client
	namespace string
}

func NewTenantSet(client *goredis.Client, namespace string) *TenantSet {
	return &TenantSet{client: client, namespace: namespace}
}

func (s *TenantSet) key(t tenant.Model) string {
	return namespacedKey(s.namespace, TenantKey(t))
}

func (s *TenantSet) Add(ctx context.Context, t tenant.Model, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SAdd(ctx, s.key(t), toIfaces(members)...).Err()
}

func (s *TenantSet) Remove(ctx context.Context, t tenant.Model, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SRem(ctx, s.key(t), toIfaces(members)...).Err()
}

func (s *TenantSet) Members(ctx context.Context, t tenant.Model) ([]string, error) {
	return s.client.SMembers(ctx, s.key(t)).Result()
}
