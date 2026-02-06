package instance

import (
	"fmt"
	"sync"

	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type RouteRegistry struct {
	mutex    sync.RWMutex
	register map[uuid.UUID]map[uuid.UUID]RouteModel
}

var routeRegistry *RouteRegistry
var routeRegistryOnce sync.Once

func getRouteRegistry() *RouteRegistry {
	routeRegistryOnce.Do(func() {
		routeRegistry = &RouteRegistry{}
		routeRegistry.register = make(map[uuid.UUID]map[uuid.UUID]RouteModel)
	})
	return routeRegistry
}

func (r *RouteRegistry) AddTenant(t tenant.Model, routes []RouteModel) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var tenantRoutes map[uuid.UUID]RouteModel
	var ok bool
	if tenantRoutes, ok = r.register[t.Id()]; !ok {
		tenantRoutes = make(map[uuid.UUID]RouteModel)
		r.register[t.Id()] = tenantRoutes
	}
	for _, route := range routes {
		tenantRoutes[route.Id()] = route
	}
}

func (r *RouteRegistry) GetRoute(t tenant.Model, id uuid.UUID) (RouteModel, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if _, ok := r.register[t.Id()]; !ok {
		return RouteModel{}, false
	}
	if route, ok := r.register[t.Id()][id]; ok {
		return route, true
	}
	return RouteModel{}, false
}

func (r *RouteRegistry) GetRoutes(t tenant.Model) []RouteModel {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if tenantRoutes, ok := r.register[t.Id()]; ok {
		var routes []RouteModel
		for _, route := range tenantRoutes {
			routes = append(routes, route)
		}
		return routes
	}
	return make([]RouteModel, 0)
}

func (r *RouteRegistry) GetRouteByTransitMap(t tenant.Model, mapId _map.Id) (RouteModel, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if tenantRoutes, ok := r.register[t.Id()]; ok {
		for _, route := range tenantRoutes {
			if route.TransitMapId() == mapId {
				return route, nil
			}
		}
	}
	return RouteModel{}, fmt.Errorf("instance route not found for transit map %d", mapId)
}

func (r *RouteRegistry) IsTransitMap(t tenant.Model, mapId _map.Id) bool {
	_, err := r.GetRouteByTransitMap(t, mapId)
	return err == nil
}

func (r *RouteRegistry) ClearTenant(t tenant.Model) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	count := 0
	if tenantRoutes, ok := r.register[t.Id()]; ok {
		count = len(tenantRoutes)
		delete(r.register, t.Id())
	}
	return count
}
