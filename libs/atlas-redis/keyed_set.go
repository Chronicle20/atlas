package redis

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// KeyedSet is a family of env-global Redis SETs, one per key K (no tenant
// scoping). Key format: <prefix>:<namespace>:<keyFn(k)>. The env-global
// analogue of TenantKeyedSet, mirroring KeyedHash's env-global keying.
type KeyedSet[K comparable] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
}

func NewKeyedSet[K comparable](client *goredis.Client, namespace string, keyFn func(K) string) *KeyedSet[K] {
	return &KeyedSet[K]{client: client, namespace: namespace, keyFn: keyFn}
}

func (s *KeyedSet[K]) key(k K) string { return namespacedKey(s.namespace, s.keyFn(k)) }

func (s *KeyedSet[K]) Add(ctx context.Context, k K, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SAdd(ctx, s.key(k), toIfaces(members)...).Err()
}

func (s *KeyedSet[K]) Remove(ctx context.Context, k K, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SRem(ctx, s.key(k), toIfaces(members)...).Err()
}

func (s *KeyedSet[K]) Members(ctx context.Context, k K) ([]string, error) {
	return s.client.SMembers(ctx, s.key(k)).Result()
}

func (s *KeyedSet[K]) IsMember(ctx context.Context, k K, member string) (bool, error) {
	return s.client.SIsMember(ctx, s.key(k), member).Result()
}

// Clear removes the entire SET for k.
func (s *KeyedSet[K]) Clear(ctx context.Context, k K) error {
	return s.client.Del(ctx, s.key(k)).Err()
}

// ClearAll deletes every SET in this KeyedSet's namespace (SCAN COUNT=100 +
// pipelined DEL). Returns the number of keys deleted. Mirrors Registry.Clear.
func (s *KeyedSet[K]) ClearAll(ctx context.Context) (int, error) {
	pattern := namespacedKey(s.namespace, "*")
	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()

	deleted := 0
	pipe := s.client.Pipeline()
	pipeSize := 0
	var firstErr error

	flushPipe := func() {
		if pipeSize == 0 {
			return
		}
		if _, err := pipe.Exec(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		pipe = s.client.Pipeline()
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
