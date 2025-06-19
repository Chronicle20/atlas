package transport

import (
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
)

type RouteRegistry struct {
	mutex         sync.RWMutex
	stateRegister map[uuid.UUID]map[uuid.UUID]*StateMachine
	routes        map[uuid.UUID][]Model
}

var routeRegistry *RouteRegistry
var routeRegistryOnce sync.Once

func getRouteRegistry() *RouteRegistry {
	routeRegistryOnce.Do(func() {
		routeRegistry = &RouteRegistry{}
		routeRegistry.stateRegister = make(map[uuid.UUID]map[uuid.UUID]*StateMachine)
		routeRegistry.routes = make(map[uuid.UUID][]Model)
	})
	return routeRegistry
}

func (r *RouteRegistry) AddTenant(t tenant.Model, routes []Model) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var tenantStates map[uuid.UUID]*StateMachine
	var ok bool
	if tenantStates, ok = r.stateRegister[t.Id()]; !ok {
		tenantStates = make(map[uuid.UUID]*StateMachine)
		r.stateRegister[t.Id()] = tenantStates
	}
	for _, route := range routes {
		tenantStates[route.Id()] = NewStateMachine(route)
	}
	r.routes[t.Id()] = routes
}

func (r *RouteRegistry) GetRoutes(t tenant.Model) []Model {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.routes[t.Id()]
}

func (r *RouteRegistry) GetRouteStateMachine(t tenant.Model, id uuid.UUID) (*StateMachine, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var tenantStates map[uuid.UUID]*StateMachine
	var ok bool
	if tenantStates, ok = r.stateRegister[t.Id()]; !ok {
		return nil, false
	}
	var stateMachine *StateMachine
	if stateMachine, ok = tenantStates[id]; !ok {
		return nil, false
	}
	return stateMachine, true
}

func (r *RouteRegistry) GetStateMachines(t tenant.Model) []*StateMachine {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var tenantStates map[uuid.UUID]*StateMachine
	var ok bool
	if tenantStates, ok = r.stateRegister[t.Id()]; !ok {
		return nil
	}
	var stateMachines []*StateMachine
	for _, stateMachine := range tenantStates {
		stateMachines = append(stateMachines, stateMachine)
	}
	return stateMachines
}
