package transport

import (
	"atlas-transports/character"
	charactermock "atlas-transports/character/mock"
	"atlas-transports/kafka/message"
	"bytes"
	"context"
	"time"

	"testing"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/alicebob/miniredis/v2"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTransportTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRouteRegistry(rc)
}

func newTestTenantContext(t *testing.T) (tenant.Model, context.Context) {
	t.Helper()
	tenantId := uuid.New()
	tm, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)
	return tm, ctx
}

func TestRouteRegistry_GetRouteByStartMap(t *testing.T) {
	setupTransportTestRegistry(t)

	// Create test tenants
	_, ctx1 := newTestTenantContext(t)
	_, ctx2 := newTestTenantContext(t)

	// Create test routes with different start map IDs
	route1, err := NewBuilder("Ellinia to Orbis").
		SetStartMapId(_map.Id(101000300)).
		SetStagingMapId(_map.Id(101000301)).
		SetEnRouteMapIds([]_map.Id{_map.Id(101000302)}).
		SetDestinationMapId(_map.Id(200000100)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	route2, err := NewBuilder("Orbis to Ludibrium").
		SetStartMapId(_map.Id(200000100)).
		SetStagingMapId(_map.Id(200000110)).
		SetEnRouteMapIds([]_map.Id{_map.Id(200000111)}).
		SetDestinationMapId(_map.Id(220000000)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	route3, err := NewBuilder("Different Tenant Route").
		SetStartMapId(_map.Id(101000300)). // Same start map as route1
		SetStagingMapId(_map.Id(101000301)).
		SetEnRouteMapIds([]_map.Id{_map.Id(101000302)}).
		SetDestinationMapId(_map.Id(300000000)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	tests := []struct {
		name          string
		setup         func()
		ctx           context.Context
		mapId         _map.Id
		expectedRoute Model
		expectError   bool
	}{
		{
			name: "Successful route retrieval",
			setup: func() {
				getRouteRegistry().AddTenant(ctx1, []Model{route1, route2})
			},
			ctx:           ctx1,
			mapId:         _map.Id(101000300),
			expectedRoute: route1,
			expectError:   false,
		},
		{
			name: "Route not found",
			setup: func() {
				getRouteRegistry().AddTenant(ctx1, []Model{route1, route2})
			},
			ctx:         ctx1,
			mapId:       _map.Id(999999999),
			expectError: true,
		},
		{
			name: "Multi-tenant isolation",
			setup: func() {
				getRouteRegistry().AddTenant(ctx1, []Model{route1})
				getRouteRegistry().AddTenant(ctx2, []Model{route3})
			},
			ctx:           ctx1,
			mapId:         _map.Id(101000300),
			expectedRoute: route1,
			expectError:   false,
		},
		{
			name: "Different tenant same map ID",
			setup: func() {
				getRouteRegistry().AddTenant(ctx1, []Model{route1})
				getRouteRegistry().AddTenant(ctx2, []Model{route3})
			},
			ctx:           ctx2,
			mapId:         _map.Id(101000300),
			expectedRoute: route3,
			expectError:   false,
		},
		{
			name:        "Empty tenant registry",
			setup:       func() {},
			ctx:         ctx1,
			mapId:       _map.Id(101000300),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-initialize registry for each test
			setupTransportTestRegistry(t)

			// Setup test data
			tt.setup()

			// Execute test
			result, err := getRouteRegistry().GetRouteByStartMap(tt.ctx, tt.mapId)

			// Assert results
			if tt.expectError {
				assert.Error(t, err, "Expected an error")
				assert.Equal(t, Model{}, result, "Should return empty model on error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				assert.Equal(t, tt.expectedRoute.Id(), result.Id(), "Route ID should match")
				assert.Equal(t, tt.expectedRoute.Name(), result.Name(), "Route name should match")
				assert.Equal(t, tt.expectedRoute.StartMapId(), result.StartMapId(), "Start map ID should match")
			}
		})
	}
}

func TestRouteRegistry_GetRouteByStartMap_MultipleRoutesOneMap(t *testing.T) {
	setupTransportTestRegistry(t)
	_, ctx := newTestTenantContext(t)

	route1, err := NewBuilder("Route 1").
		SetStartMapId(_map.Id(101000300)).
		SetStagingMapId(_map.Id(101000301)).
		SetEnRouteMapIds([]_map.Id{_map.Id(101000302)}).
		SetDestinationMapId(_map.Id(200000100)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	route2, err := NewBuilder("Route 2").
		SetStartMapId(_map.Id(101000300)). // Same start map as route1
		SetStagingMapId(_map.Id(101000302)).
		SetEnRouteMapIds([]_map.Id{_map.Id(101000303)}).
		SetDestinationMapId(_map.Id(300000000)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)

	getRouteRegistry().AddTenant(ctx, []Model{route1, route2})

	// Should return one of the routes (order not guaranteed due to map iteration)
	result, err := getRouteRegistry().GetRouteByStartMap(ctx, _map.Id(101000300))

	assert.NoError(t, err, "Should not return an error")
	assert.Equal(t, _map.Id(101000300), result.StartMapId(), "Start map ID should match")
	// We can't assert which specific route is returned due to iteration order
	// But we can assert that it's one of the two routes
	assert.True(t,
		result.Id() == route1.Id() || result.Id() == route2.Id(),
		"Should return one of the routes with the matching start map")
}

// Helper function to create a test route
func createTestRoute(t *testing.T, name string, startMapId, stagingMapId _map.Id, enRouteMapIds []_map.Id, destinationMapId _map.Id) Model {
	route, err := NewBuilder(name).
		SetStartMapId(startMapId).
		SetStagingMapId(stagingMapId).
		SetEnRouteMapIds(enRouteMapIds).
		SetDestinationMapId(destinationMapId).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	require.NoError(t, err)
	return route
}

// Helper to create a ProcessorImpl with mock character processor for testing
func createTestProcessor(tenantModel tenant.Model, charP character.Processor) *ProcessorImpl {
	l := logrus.New()
	l.SetOutput(&bytes.Buffer{})
	ctx := tenant.WithContext(context.Background(), tenantModel)

	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		t:     tenantModel,
		charP: charP,
	}
}

func TestWarpToRouteStartMapOnLogout_FromStagingMap(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)
	getRouteRegistry().AddTenant(ctx, []Model{route})

	var warpedCharacterId uint32
	var warpedToFieldId field.Id

	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpedCharacterId = characterId
					warpedToFieldId = fieldId
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, stagingMapId).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	assert.Equal(t, uint32(12345), warpedCharacterId, "Character ID should match")
	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, startMapId, targetField.MapId(), "Should warp to start map")
}

func TestWarpToRouteStartMapOnLogout_FromEnRouteMap(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry EnRoute", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)
	getRouteRegistry().AddTenant(ctx, []Model{route})

	var warpedCharacterId uint32
	var warpedToFieldId field.Id

	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpedCharacterId = characterId
					warpedToFieldId = fieldId
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, enRouteMapId).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	assert.Equal(t, uint32(12345), warpedCharacterId, "Character ID should match")
	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, startMapId, targetField.MapId(), "Should warp to start map")
}

func TestWarpToRouteStartMapOnLogout_FromDestinationMap_NoWarp(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry Dest", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)
	getRouteRegistry().AddTenant(ctx, []Model{route})

	warpCalled := false
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpCalled = true
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, destinationMapId).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	assert.False(t, warpCalled, "Should not warp from destination map")
}

func TestWarpToRouteStartMapOnLogout_FromUnrelatedMap_NoWarp(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry Unrelated", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)
	getRouteRegistry().AddTenant(ctx, []Model{route})

	warpCalled := false
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpCalled = true
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	unrelatedMapId := _map.Id(100000000)
	currentField := field.NewBuilder(0, 0, unrelatedMapId).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	assert.False(t, warpCalled, "Should not warp from unrelated map")
}

func TestWarpToRouteStartMapOnLogout_MultipleRoutes_CorrectMatch(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tenantModel)

	route1 := createTestRoute(t, "Ferry 1",
		_map.Id(101000300),
		_map.Id(200090000),
		[]_map.Id{_map.Id(200090100)},
		_map.Id(200000100))

	route2 := createTestRoute(t, "Ferry 2",
		_map.Id(220000000),
		_map.Id(220090000),
		[]_map.Id{_map.Id(220090100)},
		_map.Id(220000100))

	getRouteRegistry().AddTenant(ctx, []Model{route1, route2})

	var warpedToFieldId field.Id
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpedToFieldId = fieldId
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, _map.Id(220090000)).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, _map.Id(220000000), targetField.MapId(), "Should warp to route2's start map")
}

func TestWarpToRouteStartMapOnLogout_NoRoutes_NoError(t *testing.T) {
	setupTransportTestRegistry(t)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	warpCalled := false
	mockCharP := &charactermock.ProcessorMock{
		WarpRandomFunc: func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
			return func(characterId uint32) func(fieldId field.Id) error {
				return func(fieldId field.Id) error {
					warpCalled = true
					return nil
				}
			}
		},
	}

	processor := createTestProcessor(tenantModel, mockCharP)
	currentField := field.NewBuilder(0, 0, _map.Id(100000000)).Build()
	mb := &message.Buffer{}

	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)

	require.NoError(t, err)
	assert.False(t, warpCalled, "Should not warp when no routes exist")
}
