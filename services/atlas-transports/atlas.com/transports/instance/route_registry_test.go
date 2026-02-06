package instance

import (
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newTestRouteRegistry() *RouteRegistry {
	return &RouteRegistry{
		register: make(map[uuid.UUID]map[uuid.UUID]RouteModel),
	}
}

func newTestTenantModel(t *testing.T) tenant.Model {
	tenantId := uuid.New()
	tm, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)
	return tm
}

func TestRouteRegistry_AddAndGet(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)
	route := newTestRoute()

	reg.AddTenant(tm, []RouteModel{route})

	got, ok := reg.GetRoute(tm, route.Id())
	assert.True(t, ok)
	assert.Equal(t, route.Name(), got.Name())
}

func TestRouteRegistry_GetRoute_NotFound(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)

	_, ok := reg.GetRoute(tm, uuid.New())
	assert.False(t, ok)
}

func TestRouteRegistry_GetRoutes(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)

	route1, _ := NewRouteBuilder("route1").
		SetTransitMapIds([]_map.Id{100}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()
	route2, _ := NewRouteBuilder("route2").
		SetTransitMapIds([]_map.Id{200}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()

	reg.AddTenant(tm, []RouteModel{route1, route2})

	routes := reg.GetRoutes(tm)
	assert.Len(t, routes, 2)
}

func TestRouteRegistry_GetRoutes_EmptyTenant(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)

	routes := reg.GetRoutes(tm)
	assert.Len(t, routes, 0)
}

func TestRouteRegistry_GetRouteByTransitMap(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)
	route := newTestRoute()

	reg.AddTenant(tm, []RouteModel{route})

	got, err := reg.GetRouteByTransitMap(tm, route.TransitMapIds()[0])
	assert.NoError(t, err)
	assert.Equal(t, route.Id(), got.Id())
}

func TestRouteRegistry_GetRouteByTransitMap_NotFound(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)

	_, err := reg.GetRouteByTransitMap(tm, _map.Id(999999))
	assert.Error(t, err)
}

func TestRouteRegistry_IsTransitMap(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)
	route := newTestRoute()

	reg.AddTenant(tm, []RouteModel{route})

	assert.True(t, reg.IsTransitMap(tm, route.TransitMapIds()[0]))
	assert.False(t, reg.IsTransitMap(tm, _map.Id(999999)))
}

func TestRouteRegistry_ClearTenant(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)
	route := newTestRoute()

	reg.AddTenant(tm, []RouteModel{route})
	count := reg.ClearTenant(tm)

	assert.Equal(t, 1, count)
	routes := reg.GetRoutes(tm)
	assert.Len(t, routes, 0)
}

func TestRouteRegistry_ClearTenant_Empty(t *testing.T) {
	reg := newTestRouteRegistry()
	tm := newTestTenantModel(t)

	count := reg.ClearTenant(tm)
	assert.Equal(t, 0, count)
}
