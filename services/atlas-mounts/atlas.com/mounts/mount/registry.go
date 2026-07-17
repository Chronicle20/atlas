package mount

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// MountRideContext captures the per-character state needed to tick an active
// (tamed) mount: the world it is in (for status event emission) plus the skill
// and vehicle ids that identify the mount.
type MountRideContext struct {
	WorldId   world.Id
	SkillId   int32
	VehicleId int32
}

// ActiveEntry is a single active mount, carrying enough context for the ticker
// to iterate all active mounts across tenants and emit per-tenant status events.
type ActiveEntry struct {
	Tenant      tenant.Model
	CharacterId uint32
	Ctx         MountRideContext
}

type Registry struct {
	reg     *atlas.TenantRegistry[uint32, MountRideContext]
	tenants *atlas.Set
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, MountRideContext](client, "mount-active", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		tenants: atlas.NewSet(client, "mount-active:_tenants"),
	}
}

func GetRegistry() *Registry { return registry }

// Add records (or overwrites) the active mount for a character within the
// tenant carried by ctx.
func (r *Registry) Add(ctx context.Context, characterId uint32, c MountRideContext) error {
	t := tenant.MustFromContext(ctx)
	if err := r.reg.Put(ctx, t, characterId, c); err != nil {
		return err
	}
	if tb, err := json.Marshal(&t); err == nil {
		_ = r.tenants.Add(ctx, string(tb))
	}
	return nil
}

// Remove clears the active mount for a character within the tenant carried by
// ctx.
func (r *Registry) Remove(ctx context.Context, characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	return r.reg.Remove(ctx, t, characterId)
}

// GetActive returns every active mount across all tenants. The ticker uses this
// to iterate all tamed mounts and emit per-tenant TICK/status events.
func (r *Registry) GetActive(ctx context.Context) ([]ActiveEntry, error) {
	result := make([]ActiveEntry, 0)

	members, err := r.tenants.Members(ctx)
	if err != nil {
		return result, err
	}

	for _, m := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(m), &t); err != nil {
			continue
		}

		entries, err := r.reg.GetAllEntries(ctx, t)
		if err != nil {
			continue
		}

		for charIdStr, c := range entries {
			charId, err := strconv.ParseUint(charIdStr, 10, 32)
			if err != nil {
				continue
			}
			result = append(result, ActiveEntry{
				Tenant:      t,
				CharacterId: uint32(charId),
				Ctx:         c,
			})
		}
	}
	return result, nil
}
