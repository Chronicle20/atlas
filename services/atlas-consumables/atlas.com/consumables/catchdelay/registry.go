// Package catchdelay enforces a catch item's WZ useDelay server-side. The delay
// is server-enforced because BRIDLE_MOB_CATCH_FAIL reason 1 renders the item's
// delayMsg (design §6.4) — without enforcement, reason 1 would be unreachable
// and delayMsg dead data.
package catchdelay

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type key struct {
	characterId uint32
	itemId      uint32
}

type Registry struct {
	reg *atlas.TenantRegistry[key, bool]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[key, bool](client, "consumable-catch-delay", func(k key) string {
			return fmt.Sprintf("%d:%d", k.characterId, k.itemId)
		}),
	}
}

func GetRegistry() *Registry { return registry }

// Allow reports whether a catch attempt may proceed and, when it may, arms the
// cooldown. The window is armed on EVERY admitted attempt, success or failure —
// the client's own 200ms ExclRequest floor is a separate concern and is not
// replicated here. A zero delay always admits and arms nothing.
func (r *Registry) Allow(ctx context.Context, characterId uint32, itemId uint32, delay time.Duration) (bool, error) {
	if r == nil || delay <= 0 {
		return true, nil
	}
	t := tenant.MustFromContext(ctx)
	k := key{characterId: characterId, itemId: itemId}

	exists, err := r.reg.Exists(ctx, t, k)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := r.reg.PutWithTTL(ctx, t, k, true, delay); err != nil {
		return false, err
	}
	return true, nil
}
