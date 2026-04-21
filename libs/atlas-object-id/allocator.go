// Package objectid provides a tenant-scoped allocator for client-visible
// object IDs (the uint32 "oid" used by the v83 MapleStory client to key map
// objects). Monsters, reactors, and drops share one namespace per tenant so
// the client never sees two entities with the same oid in the same map
// instance.
//
// Scope choice: the client only requires uniqueness within a field (map
// instance), but we allocate at tenant scope for two reasons: (1) server-side
// storage in each service keys entities by (tenant, id) with no field
// component, so per-field allocation would let two entities on different
// fields collide in storage; and (2) 2B ids per tenant is far more than any
// realistic workload consumes. LIFO recycle keeps the live range small.
package objectid

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// MinId is the first value ever returned by Allocate for a fresh field.
	// Starts at 1,000,000 to stay clear of static NPC oids (assigned per-map
	// starting at 1 from the WZ data, typically under 100 per map).
	MinId = uint32(1000000)
	// MaxId is the largest value allocator will return before wrapping.
	// Chosen to stay inside positive int32 range, which matches the v83 wire
	// format for oids.
	MaxId = uint32(2147483647)
)

var ErrExhausted = errors.New("object id range exhausted for field")

// Allocator mints and recycles per-tenant object IDs shared across entity types.
type Allocator interface {
	// Allocate returns a new or recycled ID for the given tenant. Recycled IDs
	// are preferred (LIFO) so freshly killed monsters or picked-up drops reuse
	// their oid quickly.
	Allocate(ctx context.Context, t tenant.Model) (uint32, error)
	// Release returns an ID to the tenant's free list. Callers should release
	// exactly once per Allocate; calling Release twice with the same ID will
	// put it on the free list twice.
	Release(ctx context.Context, t tenant.Model, id uint32) error
	// Clear removes both the counter and free list for the tenant. Use for
	// tenant reset; callers normally should not touch this in steady state.
	Clear(ctx context.Context, t tenant.Model) error
}

type redisAllocator struct {
	client *goredis.Client
	script *goredis.Script
}

// NewRedisAllocator returns an Allocator backed by the given Redis client.
func NewRedisAllocator(client *goredis.Client) Allocator {
	// One atomic script: pop from free list if non-empty, else INCR the counter
	// (seeding it on first use). KEYS[1]=free list, KEYS[2]=counter. ARGV[1]=min,
	// ARGV[2]=max.
	script := goredis.NewScript(`
		local freeKey = KEYS[1]
		local counterKey = KEYS[2]
		local minId = tonumber(ARGV[1])
		local maxId = tonumber(ARGV[2])
		local recycled = redis.call('LPOP', freeKey)
		if recycled then
			return tonumber(recycled)
		end
		local exists = redis.call('EXISTS', counterKey)
		if exists == 0 then
			redis.call('SET', counterKey, minId)
			return minId
		end
		local val = redis.call('INCR', counterKey)
		if val > maxId then
			return -1
		end
		return val
	`)
	return &redisAllocator{client: client, script: script}
}

func counterKey(t tenant.Model) string {
	return fmt.Sprintf("atlas:oid:%s:next", t.Id().String())
}

func freeKey(t tenant.Model) string {
	return fmt.Sprintf("atlas:oid:%s:free", t.Id().String())
}

func (a *redisAllocator) Allocate(ctx context.Context, t tenant.Model) (uint32, error) {
	val, err := a.script.Run(ctx, a.client,
		[]string{freeKey(t), counterKey(t)},
		strconv.FormatUint(uint64(MinId), 10),
		strconv.FormatUint(uint64(MaxId), 10),
	).Int64()
	if err != nil {
		return 0, fmt.Errorf("allocate object id: %w", err)
	}
	if val < 0 {
		return 0, ErrExhausted
	}
	return uint32(val), nil
}

func (a *redisAllocator) Release(ctx context.Context, t tenant.Model, id uint32) error {
	return a.client.LPush(ctx, freeKey(t), strconv.FormatUint(uint64(id), 10)).Err()
}

func (a *redisAllocator) Clear(ctx context.Context, t tenant.Model) error {
	return a.client.Del(ctx, freeKey(t), counterKey(t)).Err()
}
