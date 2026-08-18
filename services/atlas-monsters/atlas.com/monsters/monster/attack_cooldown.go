package monster

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type attackCooldownRegistry struct {
	reg *atlasredis.TenantRegistry[string, int64]
}

var (
	attackCooldownReg  *attackCooldownRegistry
	attackCooldownOnce sync.Once
)

func InitAttackCooldownRegistry(rc *goredis.Client) {
	attackCooldownOnce.Do(func() {
		attackCooldownReg = &attackCooldownRegistry{
			reg: atlasredis.NewTenantRegistry[string, int64](rc, "monster-attack-cooldown", func(s string) string { return s }),
		}
	})
}

func GetAttackCooldownRegistry() *attackCooldownRegistry {
	return attackCooldownReg
}

func attackCooldownKey(monsterId uint32, attackPos uint8) string {
	return fmt.Sprintf("%s:%s",
		strconv.FormatUint(uint64(monsterId), 10),
		strconv.FormatUint(uint64(attackPos), 10),
	)
}

func attackCooldownMonsterPrefix(monsterId uint32) string {
	return fmt.Sprintf("%s:", strconv.FormatUint(uint64(monsterId), 10))
}

func (r *attackCooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, attackPos uint8) bool {
	if r == nil {
		return false
	}
	ok, err := r.reg.Exists(ctx, t, attackCooldownKey(monsterId, attackPos))
	if err != nil {
		return false
	}
	return ok
}

// SetCooldown registers a cooldown for the given (monsterId, attackPos) with
// Redis-managed TTL. A zero duration is a no-op (matches melee attacks
// where attackAfter == 0).
func (r *attackCooldownRegistry) SetCooldown(ctx context.Context, t tenant.Model, monsterId uint32, attackPos uint8, duration time.Duration) {
	if r == nil {
		return
	}
	if duration <= 0 {
		return
	}
	expiryMs := time.Now().Add(duration).UnixMilli()
	_ = r.reg.PutWithTTL(ctx, t, attackCooldownKey(monsterId, attackPos), expiryMs, duration)
}

func (r *attackCooldownRegistry) ClearCooldowns(ctx context.Context, t tenant.Model, monsterId uint32) {
	if r == nil {
		return
	}
	_, _ = r.reg.ClearByPrefix(ctx, t, attackCooldownMonsterPrefix(monsterId))
}
