package transport

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRouteRegistry_GetRouteByStartMap(t *testing.T) {
	// Create test tenants
	tenant1, _ := tenant.Register(uuid.New(), "NA", 83, 0)
	tenant2, _ := tenant.Register(uuid.New(), "NA", 83, 0)

	// Create test routes with different start map IDs
	route1 := NewBuilder("Ellinia to Orbis").
		SetStartMapId(_map.Id(101000300)).
		SetStagingMapId(_map.Id(101000301)).
		SetDestinationMapId(_map.Id(200000100)).
		Build()

	route2 := NewBuilder("Orbis to Ludibrium").
		SetStartMapId(_map.Id(200000100)).
		SetStagingMapId(_map.Id(200000110)).
		SetDestinationMapId(_map.Id(220000000)).
		Build()

	route3 := NewBuilder("Different Tenant Route").
		SetStartMapId(_map.Id(101000300)). // Same start map as route1
		SetStagingMapId(_map.Id(101000301)).
		SetDestinationMapId(_map.Id(300000000)).
		Build()

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

	route1 := NewBuilder("Route 1").
		SetStartMapId(_map.Id(101000300)).
		SetStagingMapId(_map.Id(101000301)).
		SetDestinationMapId(_map.Id(200000100)).
		Build()

	route2 := NewBuilder("Route 2").
		SetStartMapId(_map.Id(101000300)). // Same start map as route1
		SetStagingMapId(_map.Id(101000302)).
		SetDestinationMapId(_map.Id(300000000)).
		Build()

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
