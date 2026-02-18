package character

import (
	"context"
	"strconv"

	"github.com/Chronicle20/atlas-constants/field"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg *atlas.TenantRegistry[uint32, field.Model]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, field.Model](client, "consumable-map-character", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) AddCharacter(ctx context.Context, characterId uint32, f field.Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Put(ctx, t, characterId, f)
}

func (r *Registry) RemoveCharacter(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Remove(ctx, t, characterId)
}

func (r *Registry) GetMap(ctx context.Context, characterId uint32) (field.Model, bool) {
	t := tenant.MustFromContext(ctx)
	v, err := r.reg.Get(ctx, t, characterId)
	if err != nil {
		return field.Model{}, false
	}
	return v, true
}
