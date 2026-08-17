package monster

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type cooldownRegistry struct {
	reg *atlasredis.TenantRegistry[string, int64]
}

var (
	cooldownReg  *cooldownRegistry
	cooldownOnce sync.Once
)

func InitCooldownRegistry(rc *goredis.Client) {
	cooldownOnce.Do(func() {
		cooldownReg = &cooldownRegistry{
			reg: atlasredis.NewTenantRegistry[string, int64](rc, "monster-cooldown", func(s string) string { return s }),
		}
	})
}

func GetCooldownRegistry() *cooldownRegistry {
	return cooldownReg
}

func cooldownKey(monsterId uint32, skillId byte) string {
	return fmt.Sprintf("%s:%s",
		strconv.FormatUint(uint64(monsterId), 10),
		strconv.FormatUint(uint64(skillId), 10),
	)
}

func cooldownMonsterPrefix(monsterId uint32) string {
	return fmt.Sprintf("%s:", strconv.FormatUint(uint64(monsterId), 10))
}

func (r *cooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) bool {
	ok, err := r.reg.Exists(ctx, t, cooldownKey(monsterId, skillId))
	if err != nil {
		return false
	}
	return ok
}

func (r *cooldownRegistry) SetCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte, duration time.Duration) {
	expiryMs := time.Now().Add(duration).UnixMilli()
	_ = r.reg.PutWithTTL(ctx, t, cooldownKey(monsterId, skillId), expiryMs, duration)
}

// Remaining returns the time until the cooldown expires, or zero if there is
// no active cooldown. Tolerates legacy "1" values (parses to 1ms epoch =>  in
// the past => zero) and any other parse error (treats as eligible). Use
// IsOnCooldown for the simple boolean answer; Remaining is for picker
// scheduling.
func (r *cooldownRegistry) Remaining(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) time.Duration {
	expiryMs, err := r.reg.Get(ctx, t, cooldownKey(monsterId, skillId))
	if err != nil {
		if errors.Is(err, atlasredis.ErrNotFound) {
			return 0
		}
		return 0
	}
	now := time.Now().UnixMilli()
	if expiryMs <= now {
		return 0
	}
	return time.Duration(expiryMs-now) * time.Millisecond
}

func (r *cooldownRegistry) ClearCooldowns(ctx context.Context, t tenant.Model, monsterId uint32) {
	_, _ = r.reg.ClearByPrefix(ctx, t, cooldownMonsterPrefix(monsterId))
}
