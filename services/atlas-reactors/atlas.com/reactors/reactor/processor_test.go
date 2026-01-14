package reactor

import (
	"atlas-reactors/reactor/data"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Test setup helpers

func setupTestLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

func setupTestTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func setupTestContext(t tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), t)
}

// createTestReactor creates a reactor directly in the registry for testing
func createTestReactor(t tenant.Model, worldId byte, channelId byte, mapId uint32, classification uint32, name string) Model {
	builder := NewModelBuilder(t, worldId, channelId, mapId, classification, name).
		SetState(0).
		SetPosition(100, 200).
		SetDelay(0).
		SetDirection(0).
		SetData(data.Model{})
	m, _ := GetRegistry().Create(t, builder)
	return m
}

// cleanupRegistry removes all reactors for a tenant
func cleanupRegistry(t tenant.Model) {
	registry := GetRegistry()
	all := registry.GetAll()
	if reactors, ok := all[t]; ok {
		for _, r := range reactors {
			registry.Remove(t, r.Id())
		}
	}
}

// TestGetById tests the GetById processor function
func TestGetById(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(ten tenant.Model) uint32
		expectError bool
	}{
		{
			name: "success - reactor exists",
			setup: func(ten tenant.Model) uint32 {
				r := createTestReactor(ten, 1, 1, 100000, 2000000, "test-reactor")
				return r.Id()
			},
			expectError: false,
		},
		{
			name: "not found - reactor does not exist",
			setup: func(ten tenant.Model) uint32 {
				return 999999999 // Non-existent ID
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := setupTestLogger()
			ten := setupTestTenant()
			ctx := setupTestContext(ten)
			defer cleanupRegistry(ten)

			reactorId := tc.setup(ten)

			result, err := GetById(l)(ctx)(reactorId)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, reactorId, result.Id())
			}
		})
	}
}

// TestGetInMap tests the GetInMap processor function
func TestGetInMap(t *testing.T) {
	tests := []struct {
		name          string
		worldId       byte
		channelId     byte
		mapId         uint32
		setup         func(ten tenant.Model)
		expectedCount int
	}{
		{
			name:      "success - returns reactors in map",
			worldId:   1,
			channelId: 1,
			mapId:     100000,
			setup: func(ten tenant.Model) {
				createTestReactor(ten, 1, 1, 100000, 2000001, "reactor-1")
				createTestReactor(ten, 1, 1, 100000, 2000002, "reactor-2")
				createTestReactor(ten, 1, 1, 100000, 2000003, "reactor-3")
			},
			expectedCount: 3,
		},
		{
			name:      "empty - no reactors in map",
			worldId:   1,
			channelId: 1,
			mapId:     200000,
			setup: func(ten tenant.Model) {
				// Create reactors in different map
				createTestReactor(ten, 1, 1, 100000, 2000001, "reactor-1")
			},
			expectedCount: 0,
		},
		{
			name:      "filters by world",
			worldId:   2,
			channelId: 1,
			mapId:     100000,
			setup: func(ten tenant.Model) {
				createTestReactor(ten, 1, 1, 100000, 2000001, "reactor-world-1")
				createTestReactor(ten, 2, 1, 100000, 2000002, "reactor-world-2")
			},
			expectedCount: 1,
		},
		{
			name:      "filters by channel",
			worldId:   1,
			channelId: 2,
			mapId:     100000,
			setup: func(ten tenant.Model) {
				createTestReactor(ten, 1, 1, 100000, 2000001, "reactor-channel-1")
				createTestReactor(ten, 1, 2, 100000, 2000002, "reactor-channel-2")
			},
			expectedCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := setupTestLogger()
			ten := setupTestTenant()
			ctx := setupTestContext(ten)
			defer cleanupRegistry(ten)

			tc.setup(ten)

			results, err := GetInMap(l)(ctx)(tc.worldId, tc.channelId, tc.mapId)

			assert.NoError(t, err)
			assert.Len(t, results, tc.expectedCount)

			// Verify all returned reactors match the filter criteria
			for _, r := range results {
				assert.Equal(t, tc.worldId, r.WorldId())
				assert.Equal(t, tc.channelId, r.ChannelId())
				assert.Equal(t, tc.mapId, r.MapId())
			}
		})
	}
}

// TestGetInMap_MultiTenant verifies tenant isolation
func TestGetInMap_MultiTenant(t *testing.T) {
	l := setupTestLogger()

	// Setup two different tenants
	tenant1 := setupTestTenant()
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := setupTestContext(tenant1)
	ctx2 := setupTestContext(tenant2)

	defer cleanupRegistry(tenant1)
	defer cleanupRegistry(tenant2)

	// Create reactors for each tenant in same map
	createTestReactor(tenant1, 1, 1, 100000, 2000001, "tenant1-reactor")
	createTestReactor(tenant2, 1, 1, 100000, 2000002, "tenant2-reactor")

	// Query should only return reactors for the requesting tenant
	results1, err1 := GetInMap(l)(ctx1)(1, 1, 100000)
	results2, err2 := GetInMap(l)(ctx2)(1, 1, 100000)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Len(t, results1, 1)
	assert.Len(t, results2, 1)
	assert.Equal(t, "tenant1-reactor", results1[0].Name())
	assert.Equal(t, "tenant2-reactor", results2[0].Name())
}

// TestRegistry_Create tests direct registry creation
func TestRegistry_Create(t *testing.T) {
	ten := setupTestTenant()
	defer cleanupRegistry(ten)

	builder := NewModelBuilder(ten, 1, 1, 100000, 2000000, "test-reactor").
		SetState(1).
		SetPosition(150, 250).
		SetDelay(100).
		SetDirection(4)

	created, err := GetRegistry().Create(ten, builder)

	assert.NoError(t, err)
	assert.NotEqual(t, uint32(0), created.Id())
	assert.Equal(t, byte(1), created.WorldId())
	assert.Equal(t, byte(1), created.ChannelId())
	assert.Equal(t, uint32(100000), created.MapId())
	assert.Equal(t, uint32(2000000), created.Classification())
	assert.Equal(t, "test-reactor", created.Name())
	assert.Equal(t, int8(1), created.State())
	assert.Equal(t, int16(150), created.X())
	assert.Equal(t, int16(250), created.Y())
	assert.Equal(t, uint32(100), created.Delay())
	assert.Equal(t, byte(4), created.Direction())
}

// TestRegistry_Get tests direct registry get
func TestRegistry_Get(t *testing.T) {
	ten := setupTestTenant()
	defer cleanupRegistry(ten)

	// Create a reactor
	created := createTestReactor(ten, 1, 1, 100000, 2000000, "test-reactor")

	// Get it back
	retrieved, err := GetRegistry().Get(created.Id())

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), retrieved.Id())
	assert.Equal(t, created.Name(), retrieved.Name())
}

// TestRegistry_Get_NotFound tests registry get with non-existent ID
func TestRegistry_Get_NotFound(t *testing.T) {
	_, err := GetRegistry().Get(999999999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to locate reactor")
}

// TestRegistry_Remove tests direct registry removal
func TestRegistry_Remove(t *testing.T) {
	ten := setupTestTenant()
	defer cleanupRegistry(ten)

	// Create a reactor
	created := createTestReactor(ten, 1, 1, 100000, 2000000, "test-reactor")

	// Verify it exists
	_, err := GetRegistry().Get(created.Id())
	assert.NoError(t, err)

	// Remove it
	GetRegistry().Remove(ten, created.Id())

	// Verify it's gone
	_, err = GetRegistry().Get(created.Id())
	assert.Error(t, err)
}

// TestRegistry_GetInMap tests direct registry map queries
func TestRegistry_GetInMap(t *testing.T) {
	ten := setupTestTenant()
	defer cleanupRegistry(ten)

	// Create multiple reactors in same map
	createTestReactor(ten, 1, 1, 100000, 2000001, "reactor-1")
	createTestReactor(ten, 1, 1, 100000, 2000002, "reactor-2")

	results := GetRegistry().GetInMap(ten, 1, 1, 100000)

	assert.Len(t, results, 2)
}

// TestRegistry_GetAll tests direct registry get all
func TestRegistry_GetAll(t *testing.T) {
	tenant1 := setupTestTenant()
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	defer cleanupRegistry(tenant1)
	defer cleanupRegistry(tenant2)

	// Create reactors for different tenants
	createTestReactor(tenant1, 1, 1, 100000, 2000001, "tenant1-reactor-1")
	createTestReactor(tenant1, 1, 1, 100000, 2000002, "tenant1-reactor-2")
	createTestReactor(tenant2, 1, 1, 100000, 2000003, "tenant2-reactor-1")

	all := GetRegistry().GetAll()

	assert.Len(t, all[tenant1], 2)
	assert.Len(t, all[tenant2], 1)
}

// TestRegistry_Remove_FromMapIndex verifies removal updates map index
func TestRegistry_Remove_FromMapIndex(t *testing.T) {
	ten := setupTestTenant()
	defer cleanupRegistry(ten)

	// Create two reactors in same map
	reactor1 := createTestReactor(ten, 1, 1, 100000, 2000001, "reactor-1")
	reactor2 := createTestReactor(ten, 1, 1, 100000, 2000002, "reactor-2")

	// Verify both in map index
	results := GetRegistry().GetInMap(ten, 1, 1, 100000)
	assert.Len(t, results, 2)

	// Remove one
	GetRegistry().Remove(ten, reactor1.Id())

	// Verify only one remains in map index
	results = GetRegistry().GetInMap(ten, 1, 1, 100000)
	assert.Len(t, results, 1)
	assert.Equal(t, reactor2.Id(), results[0].Id())
}

// TestRegistry_UniqueIds verifies ID generation produces unique IDs
func TestRegistry_UniqueIds(t *testing.T) {
	ten := setupTestTenant()
	defer cleanupRegistry(ten)

	ids := make(map[uint32]bool)

	// Create multiple reactors and verify unique IDs
	for i := 0; i < 100; i++ {
		r := createTestReactor(ten, 1, 1, 100000, uint32(2000000+i), "reactor")
		assert.False(t, ids[r.Id()], "Duplicate ID generated: %d", r.Id())
		ids[r.Id()] = true
	}
}
