package monster

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type cooldownRegistry struct {
	client *goredis.Client
}

var cooldownReg *cooldownRegistry
var cooldownOnce sync.Once

func InitCooldownRegistry(rc *goredis.Client) {
	cooldownOnce.Do(func() {
		cooldownReg = &cooldownRegistry{client: rc}
	})
}

func GetCooldownRegistry() *cooldownRegistry {
	return cooldownReg
}

func cooldownKey(t tenant.Model, monsterId uint32, skillId byte) string {
	return fmt.Sprintf("atlas:monster-cooldown:%s:%s:%s",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
		strconv.FormatUint(uint64(skillId), 10),
	)
}

func cooldownScanPattern(t tenant.Model, monsterId uint32) string {
	return fmt.Sprintf("atlas:monster-cooldown:%s:%s:*",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
	)
}

func (r *cooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) bool {
	key := cooldownKey(t, monsterId, skillId)
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return result > 0
}

func (r *cooldownRegistry) SetCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte, duration time.Duration) {
	key := cooldownKey(t, monsterId, skillId)
	expiryMs := time.Now().Add(duration).UnixMilli()
	r.client.Set(ctx, key, strconv.FormatInt(expiryMs, 10), duration)
}

// Remaining returns the time until the cooldown expires, or zero if there is
// no active cooldown. Tolerates legacy "1" values (parses to 1ms epoch =>  in
// the past => zero) and any other parse error (treats as eligible). Use
// IsOnCooldown for the simple boolean answer; Remaining is for picker
// scheduling.
func (r *cooldownRegistry) Remaining(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) time.Duration {
	key := cooldownKey(t, monsterId, skillId)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return 0
	}
	expiryMs, perr := strconv.ParseInt(val, 10, 64)
	if perr != nil {
		return 0
	}
	now := time.Now().UnixMilli()
	if expiryMs <= now {
		return 0
	}
	return time.Duration(expiryMs-now) * time.Millisecond
}

func (r *cooldownRegistry) ClearCooldowns(ctx context.Context, t tenant.Model, monsterId uint32) {
	pattern := cooldownScanPattern(t, monsterId)
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			r.client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}
