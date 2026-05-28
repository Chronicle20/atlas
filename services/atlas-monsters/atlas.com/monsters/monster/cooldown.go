package monster

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type cooldownRegistry struct {
	reg *atlasredis.Registry[string, int64]
}

var cooldownReg *cooldownRegistry
var cooldownOnce sync.Once

func InitCooldownRegistry(rc *goredis.Client) {
	cooldownOnce.Do(func() {
		cooldownReg = &cooldownRegistry{
			reg: atlasredis.NewRegistry[string, int64](rc, "monster-cooldown", func(s string) string { return s }),
		}
	})
}

func GetCooldownRegistry() *cooldownRegistry {
	return cooldownReg
}

func cooldownSuffix(t tenant.Model, monsterId uint32, skillId byte) string {
	return fmt.Sprintf("%s:%s:%s",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
		strconv.FormatUint(uint64(skillId), 10),
	)
}

func cooldownMonsterPrefix(t tenant.Model, monsterId uint32) string {
	return fmt.Sprintf("%s:%s:",
		t.Id().String(),
		strconv.FormatUint(uint64(monsterId), 10),
	)
}

func (r *cooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) bool {
	ok, err := r.reg.Exists(ctx, cooldownSuffix(t, monsterId, skillId))
	if err != nil {
		return false
	}
	return ok
}

func (r *cooldownRegistry) SetCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte, duration time.Duration) {
	expiryMs := time.Now().Add(duration).UnixMilli()
	_ = r.reg.PutWithTTL(ctx, cooldownSuffix(t, monsterId, skillId), expiryMs, duration)
}

// Remaining returns the time until the cooldown expires, or zero if there is
// no active cooldown. Tolerates legacy "1" values (parses to 1ms epoch =>  in
// the past => zero) and any other parse error (treats as eligible). Use
// IsOnCooldown for the simple boolean answer; Remaining is for picker
// scheduling.
func (r *cooldownRegistry) Remaining(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) time.Duration {
	expiryMs, err := r.reg.Get(ctx, cooldownSuffix(t, monsterId, skillId))
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
	_, _ = r.reg.ClearByPrefix(ctx, cooldownMonsterPrefix(t, monsterId))
}
