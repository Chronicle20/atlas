package monster

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-tenant"
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

func cooldownKey(t tenant.Model, monsterId uint32, skillId uint16) string {
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

func (r *cooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId uint16) bool {
	key := cooldownKey(t, monsterId, skillId)
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return result > 0
}

func (r *cooldownRegistry) SetCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId uint16, duration time.Duration) {
	key := cooldownKey(t, monsterId, skillId)
	r.client.Set(ctx, key, "1", duration)
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
