package instance

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
	routes *atlas.TenantRegistry[uuid.UUID, RouteModel]
}

var routeRegistry *RouteRegistry

func InitRouteRegistry(client *goredis.Client) {
	routeRegistry = &RouteRegistry{
		routes: atlas.NewTenantRegistry[uuid.UUID, RouteModel](client, "instance-route", func(id uuid.UUID) string {
			return id.String()
		}),
	}
}

func getRouteRegistry() *RouteRegistry {
	return routeRegistry
}

func (r *RouteRegistry) AddTenant(ctx context.Context, routes []RouteModel) {
	t := tenant.MustFromContext(ctx)
	for _, route := range routes {
		_ = r.routes.Put(ctx, t, route.Id(), route)
	}
}

func (r *RouteRegistry) GetRoute(ctx context.Context, id uuid.UUID) (RouteModel, bool) {
	t := tenant.MustFromContext(ctx)
	route, err := r.routes.Get(ctx, t, id)
	if err != nil {
		return RouteModel{}, false
	}
	return route, true
}

func (r *RouteRegistry) GetRoutes(ctx context.Context) []RouteModel {
	t := tenant.MustFromContext(ctx)
	routes, err := r.routes.GetAllValues(ctx, t)
	if err != nil {
		return make([]RouteModel, 0)
	}
	return routes
}

func (r *RouteRegistry) GetRouteByTransitMap(ctx context.Context, mapId _map.Id) (RouteModel, error) {
	routes := r.GetRoutes(ctx)
	for _, route := range routes {
		if route.HasTransitMap(mapId) {
			return route, nil
		}
	}
	return RouteModel{}, fmt.Errorf("instance route not found for transit map %d", mapId)
}

func (r *RouteRegistry) IsTransitMap(ctx context.Context, mapId _map.Id) bool {
	_, err := r.GetRouteByTransitMap(ctx, mapId)
	return err == nil
}

func (r *RouteRegistry) ClearTenant(ctx context.Context) int {
	routes := r.GetRoutes(ctx)
	count := len(routes)
	t := tenant.MustFromContext(ctx)
	for _, route := range routes {
		_ = r.routes.Remove(ctx, t, route.Id())
	}
	return count
}
