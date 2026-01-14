package transport

import (
	"atlas-transports/character"
	charactermock "atlas-transports/character/mock"
	"atlas-transports/kafka/message"
	"bytes"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRouteRegistry_GetRouteByStartMap(t *testing.T) {
	// Create test tenants
	tenant1, _ := tenant.Register(uuid.New(), "NA", 83, 0)
	tenant2, _ := tenant.Register(uuid.New(), "NA", 83, 0)

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
		setup         func(*RouteRegistry)
		tenant        tenant.Model
		mapId         _map.Id
		expectedRoute Model
		expectError   bool
	}{
		{
			name: "Successful route retrieval",
			setup: func(registry *RouteRegistry) {
				registry.AddTenant(tenant1, []Model{route1, route2})
			},
			tenant:        tenant1,
			mapId:         _map.Id(101000300),
			expectedRoute: route1,
			expectError:   false,
		},
		{
			name: "Route not found",
			setup: func(registry *RouteRegistry) {
				registry.AddTenant(tenant1, []Model{route1, route2})
			},
			tenant:      tenant1,
			mapId:       _map.Id(999999999),
			expectError: true,
		},
		{
			name: "Multi-tenant isolation",
			setup: func(registry *RouteRegistry) {
				registry.AddTenant(tenant1, []Model{route1})
				registry.AddTenant(tenant2, []Model{route3})
			},
			tenant:        tenant1,
			mapId:         _map.Id(101000300),
			expectedRoute: route1,
			expectError:   false,
		},
		{
			name: "Different tenant same map ID",
			setup: func(registry *RouteRegistry) {
				registry.AddTenant(tenant1, []Model{route1})
				registry.AddTenant(tenant2, []Model{route3})
			},
			tenant:        tenant2,
			mapId:         _map.Id(101000300),
			expectedRoute: route3,
			expectError:   false,
		},
		{
			name: "Empty tenant registry",
			setup: func(registry *RouteRegistry) {
				// Don't add any routes
			},
			tenant:      tenant1,
			mapId:       _map.Id(101000300),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh registry for each test
			registry := &RouteRegistry{
				routeRegister: make(map[uuid.UUID]map[uuid.UUID]Model),
			}

			// Setup test data
			tt.setup(registry)

			// Execute test
			result, err := registry.GetRouteByStartMap(tt.tenant, tt.mapId)

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
	// This test documents the assumption: one route per start map
	// If multiple routes have the same start map, the first one found is returned
	tenant1, _ := tenant.Register(uuid.New(), "NA", 83, 0)

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

	registry := &RouteRegistry{
		routeRegister: make(map[uuid.UUID]map[uuid.UUID]Model),
	}
	registry.AddTenant(tenant1, []Model{route1, route2})

	// Should return one of the routes (order not guaranteed due to map iteration)
	result, err := registry.GetRouteByStartMap(tenant1, _map.Id(101000300))

	assert.NoError(t, err, "Should not return an error")
	assert.Equal(t, _map.Id(101000300), result.StartMapId(), "Start map ID should match")
	// We can't assert which specific route is returned due to map iteration order
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
func createTestProcessor(t *testing.T, tenantModel tenant.Model, charP character.Processor) *ProcessorImpl {
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
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	// Create test route
	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)

	// Add to the global registry (used by AllRoutesProvider)
	getRouteRegistry().AddTenant(tenantModel, []Model{route})

	// Track warp calls
	var warpedCharacterId uint32
	var warpedToFieldId field.Id

	// Create mock character processor
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

	// Create processor with mock
	processor := createTestProcessor(t, tenantModel, mockCharP)

	// Create a field representing character's current location (staging map)
	currentField := field.NewBuilder(0, 0, stagingMapId).Build()

	// Create message buffer
	mb := &message.Buffer{}

	// Call WarpToRouteStartMapOnLogout
	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	// Verify warp was called with correct parameters
	assert.Equal(t, uint32(12345), warpedCharacterId, "Character ID should match")

	// Verify the target field is the start map
	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, startMapId, targetField.MapId(), "Should warp to start map")
}

func TestWarpToRouteStartMapOnLogout_FromEnRouteMap(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	// Create test route
	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry EnRoute", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)

	// Add to the global registry (used by AllRoutesProvider)
	getRouteRegistry().AddTenant(tenantModel, []Model{route})

	// Track warp calls
	var warpedCharacterId uint32
	var warpedToFieldId field.Id

	// Create mock character processor
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

	// Create processor with mock
	processor := createTestProcessor(t, tenantModel, mockCharP)

	// Create a field representing character's current location (en-route map)
	currentField := field.NewBuilder(0, 0, enRouteMapId).Build()

	// Create message buffer
	mb := &message.Buffer{}

	// Call WarpToRouteStartMapOnLogout
	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	// Verify warp was called with correct parameters
	assert.Equal(t, uint32(12345), warpedCharacterId, "Character ID should match")

	// Verify the target field is the start map
	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, startMapId, targetField.MapId(), "Should warp to start map")
}

func TestWarpToRouteStartMapOnLogout_FromDestinationMap_NoWarp(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	// Create test route
	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry Dest", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)

	// Add to the global registry (used by AllRoutesProvider)
	getRouteRegistry().AddTenant(tenantModel, []Model{route})

	// Track warp calls
	warpCalled := false

	// Create mock character processor
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

	// Create processor with mock
	processor := createTestProcessor(t, tenantModel, mockCharP)

	// Create a field representing character's current location (destination map - NOT staging/en-route)
	currentField := field.NewBuilder(0, 0, destinationMapId).Build()

	// Create message buffer
	mb := &message.Buffer{}

	// Call WarpToRouteStartMapOnLogout
	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	// Verify warp was NOT called (destination map is not in staging/en-route maps)
	assert.False(t, warpCalled, "Should not warp from destination map")
}

func TestWarpToRouteStartMapOnLogout_FromUnrelatedMap_NoWarp(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	// Create test route
	startMapId := _map.Id(101000300)
	stagingMapId := _map.Id(200090000)
	enRouteMapId := _map.Id(200090100)
	destinationMapId := _map.Id(200000100)

	route := createTestRoute(t, "Test Ferry Unrelated", startMapId, stagingMapId, []_map.Id{enRouteMapId}, destinationMapId)

	// Add to the global registry (used by AllRoutesProvider)
	getRouteRegistry().AddTenant(tenantModel, []Model{route})

	// Track warp calls
	warpCalled := false

	// Create mock character processor
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

	// Create processor with mock
	processor := createTestProcessor(t, tenantModel, mockCharP)

	// Create a field representing character's current location (completely unrelated map)
	unrelatedMapId := _map.Id(100000000)
	currentField := field.NewBuilder(0, 0, unrelatedMapId).Build()

	// Create message buffer
	mb := &message.Buffer{}

	// Call WarpToRouteStartMapOnLogout
	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	// Verify warp was NOT called
	assert.False(t, warpCalled, "Should not warp from unrelated map")
}

func TestWarpToRouteStartMapOnLogout_MultipleRoutes_CorrectMatch(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	// Create two test routes with different staging maps
	route1 := createTestRoute(t, "Ferry 1",
		_map.Id(101000300), // start
		_map.Id(200090000), // staging
		[]_map.Id{_map.Id(200090100)}, // en-route
		_map.Id(200000100)) // destination

	route2 := createTestRoute(t, "Ferry 2",
		_map.Id(220000000), // start (different)
		_map.Id(220090000), // staging (different)
		[]_map.Id{_map.Id(220090100)}, // en-route (different)
		_map.Id(220000100)) // destination

	// Add to the global registry (used by AllRoutesProvider)
	getRouteRegistry().AddTenant(tenantModel, []Model{route1, route2})

	// Track warp calls
	var warpedToFieldId field.Id

	// Create mock character processor
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

	// Create processor with mock
	processor := createTestProcessor(t, tenantModel, mockCharP)

	// Test logout from route2's staging map
	currentField := field.NewBuilder(0, 0, _map.Id(220090000)).Build()

	// Create message buffer
	mb := &message.Buffer{}

	// Call WarpToRouteStartMapOnLogout
	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)
	require.NoError(t, err)

	// Verify warp was to route2's start map (not route1's)
	targetField, ok := field.FromId(warpedToFieldId)
	assert.True(t, ok, "Should be able to parse target field ID")
	assert.Equal(t, _map.Id(220000000), targetField.MapId(), "Should warp to route2's start map")
}

func TestWarpToRouteStartMapOnLogout_NoRoutes_NoError(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	// Don't add any routes to the registry

	// Track warp calls
	warpCalled := false

	// Create mock character processor
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

	// Create processor with mock
	processor := createTestProcessor(t, tenantModel, mockCharP)

	// Create a field representing character's current location
	currentField := field.NewBuilder(0, 0, _map.Id(100000000)).Build()

	// Create message buffer
	mb := &message.Buffer{}

	// Call WarpToRouteStartMapOnLogout
	warpFn := processor.WarpToRouteStartMapOnLogout(mb)
	err = warpFn(12345, currentField)

	// Should not error, just return nil
	require.NoError(t, err)
	assert.False(t, warpCalled, "Should not warp when no routes exist")
}
