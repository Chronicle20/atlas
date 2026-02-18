package chair

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg *atlas.TenantRegistry[uint32, Model]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, Model](client, "chair", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	v, err := r.reg.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, false
	}
	return v, true
}

func (r *Registry) Set(ctx context.Context, characterId uint32, value Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Put(ctx, t, characterId, value)
}

func (r *Registry) Clear(ctx context.Context, characterId uint32) bool {
	t := tenant.MustFromContext(ctx)
	exists, _ := r.reg.Exists(ctx, t, characterId)
	if exists {
		_ = r.reg.Remove(ctx, t, characterId)
		return true
	}
	return false
}
