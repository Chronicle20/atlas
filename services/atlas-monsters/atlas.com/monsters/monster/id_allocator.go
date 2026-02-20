package monster

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// MinMonsterId is the minimum unique ID for monsters.
	MinMonsterId = uint32(1000000000)
	// MaxMonsterId is the maximum unique ID for monsters.
	MaxMonsterId = uint32(2000000000)
)

// IdAllocator manages unique monster ID allocation using Redis.
// It provides atomic allocation via INCR and recycled ID pool via a Redis list.
type IdAllocator struct {
	client *goredis.Client
}

var idAllocator *IdAllocator
var idAllocatorOnce sync.Once

func InitIdAllocator(rc *goredis.Client) {
	idAllocatorOnce.Do(func() {
		idAllocator = &IdAllocator{client: rc}
	})
}

func GetIdAllocator() *IdAllocator {
	return idAllocator
}

func idCounterKey(t tenant.Model) string {
	return fmt.Sprintf("atlas:monster-ids:%s:next", t.String())
}

func idFreeListKey(t tenant.Model) string {
	return fmt.Sprintf("atlas:monster-ids:%s:free", t.String())
}

// Allocate returns the next available monster ID for the given tenant.
// It prefers recycled IDs (LIFO via LPUSH/LPOP) over new sequential IDs.
func (a *IdAllocator) Allocate(ctx context.Context, t tenant.Model) uint32 {
	freeKey := idFreeListKey(t)

	// Try to pop a recycled ID first (LPOP = LIFO with LPUSH)
	result, err := a.client.LPop(ctx, freeKey).Result()
	if err == nil {
		id, parseErr := strconv.ParseUint(result, 10, 32)
		if parseErr == nil {
			return uint32(id)
		}
	}

	// No recycled IDs available — allocate sequentially
	counterKey := idCounterKey(t)

	// INCR is atomic; initialize to MinMonsterId if key doesn't exist
	// We use a Lua script to atomically check-and-init + increment
	script := goredis.NewScript(`
		local key = KEYS[1]
		local min = tonumber(ARGV[1])
		local max = tonumber(ARGV[2])
		local exists = redis.call('EXISTS', key)
		if exists == 0 then
			redis.call('SET', key, min)
			return min
		end
		local val = redis.call('INCR', key)
		if val > max then
			redis.call('SET', key, min)
			return min
		end
		return val
	`)

	val, err := script.Run(ctx, a.client, []string{counterKey},
		strconv.FormatUint(uint64(MinMonsterId), 10),
		strconv.FormatUint(uint64(MaxMonsterId), 10),
	).Int64()
	if err != nil {
		// Fallback — should not happen in normal operation
		return MinMonsterId
	}
	return uint32(val)
}

// Release returns a monster ID to the free pool for reuse.
func (a *IdAllocator) Release(ctx context.Context, t tenant.Model, id uint32) {
	freeKey := idFreeListKey(t)
	a.client.LPush(ctx, freeKey, strconv.FormatUint(uint64(id), 10))
}
