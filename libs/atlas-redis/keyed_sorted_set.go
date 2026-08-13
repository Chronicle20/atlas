package redis

import (
	"context"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TenantKeyedSortedSet is a family of tenant-scoped Redis SORTED SETs, one ZSET
// per key K. Members are ordered by a caller-supplied float64 score (e.g. an
// insertion timestamp). Key format: <prefix>:<namespace>:<tenantKey>:<keyFn(k)>.
type TenantKeyedSortedSet[K comparable] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
}

func NewTenantKeyedSortedSet[K comparable](client *goredis.Client, namespace string, keyFn func(K) string) *TenantKeyedSortedSet[K] {
	return &TenantKeyedSortedSet[K]{client: client, namespace: namespace, keyFn: keyFn}
}

func (s *TenantKeyedSortedSet[K]) key(t tenant.Model, k K) string {
	return tenantEntityKey(s.namespace, t, s.keyFn(k))
}

// Add inserts member with the given score (or updates its score if present).
func (s *TenantKeyedSortedSet[K]) Add(ctx context.Context, t tenant.Model, k K, member string, score float64) error {
	return s.client.ZAdd(ctx, s.key(t, k), goredis.Z{Score: score, Member: member}).Err()
}

// Remove removes member from the sorted set.
func (s *TenantKeyedSortedSet[K]) Remove(ctx context.Context, t tenant.Model, k K, member string) error {
	return s.client.ZRem(ctx, s.key(t, k), member).Err()
}

// Range returns all members ordered by score ascending.
func (s *TenantKeyedSortedSet[K]) Range(ctx context.Context, t tenant.Model, k K) ([]string, error) {
	return s.client.ZRange(ctx, s.key(t, k), 0, -1).Result()
}

// Count returns the number of members.
func (s *TenantKeyedSortedSet[K]) Count(ctx context.Context, t tenant.Model, k K) (int64, error) {
	return s.client.ZCard(ctx, s.key(t, k)).Result()
}

// Clear removes the entire sorted set for (t, k).
func (s *TenantKeyedSortedSet[K]) Clear(ctx context.Context, t tenant.Model, k K) error {
	return s.client.Del(ctx, s.key(t, k)).Err()
}

// AddBounded inserts member with the given score and enforces the buffer
// bounds in the same pipeline: members with score < minScore are pruned
// (age window), the set is trimmed to the newest maxCount members, and the
// key TTL is refreshed to ttl so idle keys evaporate without a sweeper.
// maxCount <= 0 skips the count trim; ttl <= 0 skips the TTL refresh.
func (s *TenantKeyedSortedSet[K]) AddBounded(ctx context.Context, t tenant.Model, k K, member string, score float64, minScore float64, maxCount int64, ttl time.Duration) error {
	key := s.key(t, k)
	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, key, goredis.Z{Score: score, Member: member})
	pipe.ZRemRangeByScore(ctx, key, "-inf", "("+strconv.FormatFloat(minScore, 'f', -1, 64))
	if maxCount > 0 {
		pipe.ZRemRangeByRank(ctx, key, 0, -(maxCount + 1))
	}
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}
