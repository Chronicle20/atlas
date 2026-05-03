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

type attackCooldownRegistry struct {
	client *goredis.Client
}

var attackCooldownReg *attackCooldownRegistry
var attackCooldownOnce sync.Once

func InitAttackCooldownRegistry(rc *goredis.Client) {
	attackCooldownOnce.Do(func() {
		attackCooldownReg = &attackCooldownRegistry{client: rc}
	})
}

func GetAttackCooldownRegistry() *attackCooldownRegistry {
	return attackCooldownReg
}

func attackCooldownKey(t tenant.Model, monsterId uint32, attackPos uint8) string {
	return fmt.Sprintf("atlas:monster-attack-cooldown:%s:%s:%s",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
		strconv.FormatUint(uint64(attackPos), 10),
	)
}

func attackCooldownScanPattern(t tenant.Model, monsterId uint32) string {
	return fmt.Sprintf("atlas:monster-attack-cooldown:%s:%s:*",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
	)
}

func (r *attackCooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, attackPos uint8) bool {
	key := attackCooldownKey(t, monsterId, attackPos)
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return result > 0
}

// SetCooldown registers a cooldown for the given (monsterId, attackPos) with
// Redis-managed TTL. A zero duration is a no-op (matches melee attacks
// where attackAfter == 0).
func (r *attackCooldownRegistry) SetCooldown(ctx context.Context, t tenant.Model, monsterId uint32, attackPos uint8, duration time.Duration) {
	if duration <= 0 {
		return
	}
	key := attackCooldownKey(t, monsterId, attackPos)
	expiryMs := time.Now().Add(duration).UnixMilli()
	r.client.Set(ctx, key, strconv.FormatInt(expiryMs, 10), duration)
}

func (r *attackCooldownRegistry) ClearCooldowns(ctx context.Context, t tenant.Model, monsterId uint32) {
	pattern := attackCooldownScanPattern(t, monsterId)
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
