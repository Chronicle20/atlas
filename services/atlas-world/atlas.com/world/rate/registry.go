package rate

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	rates *atlas.TenantRegistry[world.Id, Model]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		rates: atlas.NewTenantRegistry[world.Id, Model](client, "rate", func(id world.Id) string {
			return fmt.Sprintf("%d", id)
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) GetWorldRates(ctx context.Context, worldId world.Id) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.rates.Get(ctx, t, worldId)
	if err != nil {
		return NewModel()
	}
	return m
}

func (r *Registry) SetWorldRate(ctx context.Context, worldId world.Id, rateType Type, multiplier float64) Model {
	t := tenant.MustFromContext(ctx)
	current, err := r.rates.Get(ctx, t, worldId)
	if err != nil {
		current = NewModel()
	}
	current = current.WithRate(rateType, multiplier)
	_ = r.rates.Put(ctx, t, worldId, current)
	return current
}

func (r *Registry) InitWorldRates(ctx context.Context, worldId world.Id, rates Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.rates.Put(ctx, t, worldId, rates)
}
