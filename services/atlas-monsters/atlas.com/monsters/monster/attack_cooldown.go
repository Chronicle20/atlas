package monster

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type attackCooldownRegistry struct {
	reg *atlasredis.Registry[string, int64]
}

var attackCooldownReg *attackCooldownRegistry
var attackCooldownOnce sync.Once

func InitAttackCooldownRegistry(rc *goredis.Client) {
	attackCooldownOnce.Do(func() {
		attackCooldownReg = &attackCooldownRegistry{
			reg: atlasredis.NewRegistry[string, int64](rc, "monster-attack-cooldown", func(s string) string { return s }),
		}
	})
}

func GetAttackCooldownRegistry() *attackCooldownRegistry {
	return attackCooldownReg
}

func attackCooldownSuffix(t tenant.Model, monsterId uint32, attackPos uint8) string {
	return fmt.Sprintf("%s:%s:%s",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
		strconv.FormatUint(uint64(attackPos), 10),
	)
}

func attackCooldownMonsterPrefix(t tenant.Model, monsterId uint32) string {
	return fmt.Sprintf("%s:%s:",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
	)
}

func (r *attackCooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, attackPos uint8) bool {
	if r == nil {
		return false
	}
	ok, err := r.reg.Exists(ctx, attackCooldownSuffix(t, monsterId, attackPos))
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
	_ = r.reg.PutWithTTL(ctx, attackCooldownSuffix(t, monsterId, attackPos), expiryMs, duration)
}

func (r *attackCooldownRegistry) ClearCooldowns(ctx context.Context, t tenant.Model, monsterId uint32) {
	if r == nil {
		return
	}
	_, _ = r.reg.ClearByPrefix(ctx, attackCooldownMonsterPrefix(t, monsterId))
}
