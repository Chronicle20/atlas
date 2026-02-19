package instance

import (
	"context"
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/alicebob/miniredis/v2"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupRouteTestRegistry(t *testing.T) *RouteRegistry {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRouteRegistry(rc)
	return getRouteRegistry()
}

func newTestTenantContext(t *testing.T) context.Context {
	t.Helper()
	tenantId := uuid.New()
	tm, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func TestRouteRegistry_AddAndGet(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)
	route := newTestRoute()

	reg.AddTenant(ctx, []RouteModel{route})

	got, ok := reg.GetRoute(ctx, route.Id())
	assert.True(t, ok)
	assert.Equal(t, route.Name(), got.Name())
}

func TestRouteRegistry_GetRoute_NotFound(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)

	_, ok := reg.GetRoute(ctx, uuid.New())
	assert.False(t, ok)
}

func TestRouteRegistry_GetRoutes(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)

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

	reg.AddTenant(ctx, []RouteModel{route1, route2})

	routes := reg.GetRoutes(ctx)
	assert.Len(t, routes, 2)
}

func TestRouteRegistry_GetRoutes_EmptyTenant(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)

	routes := reg.GetRoutes(ctx)
	assert.Len(t, routes, 0)
}

func TestRouteRegistry_GetRouteByTransitMap(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)
	route := newTestRoute()

	reg.AddTenant(ctx, []RouteModel{route})

	got, err := reg.GetRouteByTransitMap(ctx, route.TransitMapIds()[0])
	assert.NoError(t, err)
	assert.Equal(t, route.Id(), got.Id())
}

func TestRouteRegistry_GetRouteByTransitMap_NotFound(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)

	_, err := reg.GetRouteByTransitMap(ctx, _map.Id(999999))
	assert.Error(t, err)
}

func TestRouteRegistry_IsTransitMap(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)
	route := newTestRoute()

	reg.AddTenant(ctx, []RouteModel{route})

	assert.True(t, reg.IsTransitMap(ctx, route.TransitMapIds()[0]))
	assert.False(t, reg.IsTransitMap(ctx, _map.Id(999999)))
}

func TestRouteRegistry_ClearTenant(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)
	route := newTestRoute()

	reg.AddTenant(ctx, []RouteModel{route})
	count := reg.ClearTenant(ctx)

	assert.Equal(t, 1, count)
	routes := reg.GetRoutes(ctx)
	assert.Len(t, routes, 0)
}

func TestRouteRegistry_ClearTenant_Empty(t *testing.T) {
	reg := setupRouteTestRegistry(t)
	ctx := newTestTenantContext(t)

	count := reg.ClearTenant(ctx)
	assert.Equal(t, 0, count)
}
