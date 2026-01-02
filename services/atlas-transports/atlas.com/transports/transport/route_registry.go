package transport

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"fmt"
	"github.com/google/uuid"
	"sync"
)

type RouteRegistry struct {
	mutex         sync.RWMutex
	routeRegister map[uuid.UUID]map[uuid.UUID]Model
}

var routeRegistry *RouteRegistry
var routeRegistryOnce sync.Once

func getRouteRegistry() *RouteRegistry {
	routeRegistryOnce.Do(func() {
		routeRegistry = &RouteRegistry{}
		routeRegistry.routeRegister = make(map[uuid.UUID]map[uuid.UUID]Model)
	})
	return routeRegistry
}

func (r *RouteRegistry) AddTenant(t tenant.Model, routes []Model) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var tenantStates map[uuid.UUID]Model
	var ok bool
	if tenantStates, ok = r.routeRegister[t.Id()]; !ok {
		tenantStates = make(map[uuid.UUID]Model)
		r.routeRegister[t.Id()] = tenantStates
	}
	for _, route := range routes {
		tenantStates[route.Id()] = route
	}
}

func (r *RouteRegistry) GetRoute(t tenant.Model, id uuid.UUID) (Model, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if _, ok := r.routeRegister[t.Id()]; !ok {
		return Model{}, false
	}

	if route, ok := r.routeRegister[t.Id()][id]; ok {
		return route, true
	}
	return Model{}, false
}

func (r *RouteRegistry) GetRoutes(t tenant.Model) ([]Model, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if tenantRegister, ok := r.routeRegister[t.Id()]; ok {
		var routes []Model
		for _, route := range tenantRegister {
			routes = append(routes, route)
		}
		return routes, nil
	}
	return make([]Model, 0), nil
}

func (r *RouteRegistry) GetRouteByStartMap(t tenant.Model, mapId _map.Id) (Model, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if tenantRegister, ok := r.routeRegister[t.Id()]; ok {
		for _, route := range tenantRegister {
			if route.StartMapId() == mapId {
				return route, nil
			}
		}
	}
	return Model{}, fmt.Errorf("route not found for start map %d", mapId)
}

func (r *RouteRegistry) UpdateRoute(t tenant.Model, route Model) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.routeRegister[t.Id()]; !ok {
		r.routeRegister[t.Id()] = make(map[uuid.UUID]Model)
	}
	r.routeRegister[t.Id()][route.Id()] = route
	return nil
}
