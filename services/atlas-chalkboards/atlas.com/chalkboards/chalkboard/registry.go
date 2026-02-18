package chalkboard

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg *atlas.TenantRegistry[uint32, string]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, string](client, "chalkboard", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
	}
}

func getRegistry() *Registry {
	return registry
}

func (r *Registry) Get(ctx context.Context, characterId uint32) (string, bool) {
	t := tenant.MustFromContext(ctx)
	v, err := r.reg.Get(ctx, t, characterId)
	if err != nil {
		return "", false
	}
	return v, true
}

func (r *Registry) Set(ctx context.Context, characterId uint32, value string) {
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
