package transport

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	atlas "github.com/Chronicle20/atlas-redis"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type RouteRegistry struct {
	routes *atlas.TenantRegistry[uuid.UUID, Model]
}

var routeRegistry *RouteRegistry

func InitRouteRegistry(client *goredis.Client) {
	routeRegistry = &RouteRegistry{
		routes: atlas.NewTenantRegistry[uuid.UUID, Model](client, "transport-route", func(id uuid.UUID) string {
			return id.String()
		}),
	}
}

func getRouteRegistry() *RouteRegistry {
	return routeRegistry
}

func (r *RouteRegistry) AddTenant(ctx context.Context, routes []Model) {
	t := tenant.MustFromContext(ctx)
	for _, route := range routes {
		_ = r.routes.Put(ctx, t, route.Id(), route)
	}
}

func (r *RouteRegistry) GetRoute(ctx context.Context, id uuid.UUID) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	route, err := r.routes.Get(ctx, t, id)
	if err != nil {
		return Model{}, false
	}
	return route, true
}

func (r *RouteRegistry) GetRoutes(ctx context.Context) ([]Model, error) {
	t := tenant.MustFromContext(ctx)
	routes, err := r.routes.GetAllValues(ctx, t)
	if err != nil {
		return make([]Model, 0), nil
	}
	return routes, nil
}

func (r *RouteRegistry) GetRouteByStartMap(ctx context.Context, mapId _map.Id) (Model, error) {
	routes, _ := r.GetRoutes(ctx)
	for _, route := range routes {
		if route.StartMapId() == mapId {
			return route, nil
		}
	}
	return Model{}, fmt.Errorf("route not found for start map %d", mapId)
}

func (r *RouteRegistry) UpdateRoute(ctx context.Context, route Model) error {
	t := tenant.MustFromContext(ctx)
	return r.routes.Put(ctx, t, route.Id(), route)
}

func (r *RouteRegistry) ClearTenant(ctx context.Context) int {
	routes, _ := r.GetRoutes(ctx)
	count := len(routes)
	t := tenant.MustFromContext(ctx)
	for _, route := range routes {
		_ = r.routes.Remove(ctx, t, route.Id())
	}
	return count
}
