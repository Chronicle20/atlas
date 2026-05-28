package reactor

import (
	"atlas-reactors/reactor/data"
	"atlas-reactors/reactor/data/state"
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Test setup helpers

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
}

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
func createTestReactor(t tenant.Model, worldId world.Id, channelId channel.Id, mapId _map.Id, classification uint32, name string) Model {
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	builder := NewModelBuilder(t, f, classification, name).
		SetState(0).
		SetPosition(100, 200).
		SetDelay(0).
		SetDirection(0).
		SetData(data.Model{})
	m, _ := GetRegistry().Create(t, builder)
	return m
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
			setupTestRegistry(t)
			l := setupTestLogger()
			ten := setupTestTenant()
			ctx := setupTestContext(ten)

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

// TestGetInField tests the GetInField processor function
func TestGetInField(t *testing.T) {
	tests := []struct {
		name          string
		worldId       world.Id
		channelId     channel.Id
		mapId         _map.Id
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
			setupTestRegistry(t)
			l := setupTestLogger()
			ten := setupTestTenant()
			ctx := setupTestContext(ten)

			tc.setup(ten)

			f := field.NewBuilder(tc.worldId, tc.channelId, tc.mapId).Build()
			results, err := GetInField(l)(ctx)(f)

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

// TestGetInField_MultiTenant verifies tenant isolation
func TestGetInField_MultiTenant(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()

	// Setup two different tenants
	tenant1 := setupTestTenant()
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := setupTestContext(tenant1)
	ctx2 := setupTestContext(tenant2)

	// Create reactors for each tenant in same map
	createTestReactor(tenant1, 1, 1, 100000, 2000001, "tenant1-reactor")
	createTestReactor(tenant2, 1, 1, 100000, 2000002, "tenant2-reactor")

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000)).Build()

	// Query should only return reactors for the requesting tenant
	results1, err1 := GetInField(l)(ctx1)(f)
	results2, err2 := GetInField(l)(ctx2)(f)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Len(t, results1, 1)
	assert.Len(t, results2, 1)
	assert.Equal(t, "tenant1-reactor", results1[0].Name())
	assert.Equal(t, "tenant2-reactor", results2[0].Name())
}

// TestRegistry_Create tests direct registry creation
func TestRegistry_Create(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000)).Build()
	builder := NewModelBuilder(ten, f, 2000000, "test-reactor").
		SetState(1).
		SetPosition(150, 250).
		SetDelay(100).
		SetDirection(4)

	created, err := GetRegistry().Create(ten, builder)

	assert.NoError(t, err)
	assert.NotEqual(t, uint32(0), created.Id())
	assert.Equal(t, world.Id(1), created.WorldId())
	assert.Equal(t, channel.Id(1), created.ChannelId())
	assert.Equal(t, _map.Id(100000), created.MapId())
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
	setupTestRegistry(t)
	ten := setupTestTenant()

	// Create a reactor
	created := createTestReactor(ten, 1, 1, 100000, 2000000, "test-reactor")

	// Get it back
	retrieved, err := GetRegistry().Get(ten, created.Id())

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), retrieved.Id())
	assert.Equal(t, created.Name(), retrieved.Name())
}

// TestRegistry_Get_NotFound tests registry get with non-existent ID
func TestRegistry_Get_NotFound(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()
	_, err := GetRegistry().Get(ten, 999999999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to locate reactor")
}

// TestRegistry_Remove tests direct registry removal
func TestRegistry_Remove(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()

	// Create a reactor
	created := createTestReactor(ten, 1, 1, 100000, 2000000, "test-reactor")

	// Verify it exists
	_, err := GetRegistry().Get(ten, created.Id())
	assert.NoError(t, err)

	// Remove it
	GetRegistry().Remove(ten, created.Id())

	// Verify it's gone
	_, err = GetRegistry().Get(ten, created.Id())
	assert.Error(t, err)
}

// TestRegistry_GetInField tests direct registry map queries
func TestRegistry_GetInField(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()

	// Create multiple reactors in same map
	createTestReactor(ten, 1, 1, 100000, 2000001, "reactor-1")
	createTestReactor(ten, 1, 1, 100000, 2000002, "reactor-2")

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000)).Build()
	results := GetRegistry().GetInField(ten, f)

	assert.Len(t, results, 2)
}

// TestRegistry_GetAll tests direct registry get all
func TestRegistry_GetAll(t *testing.T) {
	setupTestRegistry(t)
	tenant1 := setupTestTenant()
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

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
	setupTestRegistry(t)
	ten := setupTestTenant()

	// Create two reactors in same map
	reactor1 := createTestReactor(ten, 1, 1, 100000, 2000001, "reactor-1")
	reactor2 := createTestReactor(ten, 1, 1, 100000, 2000002, "reactor-2")

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000)).Build()

	// Verify both in map index
	results := GetRegistry().GetInField(ten, f)
	assert.Len(t, results, 2)

	// Remove one
	GetRegistry().Remove(ten, reactor1.Id())

	// Verify only one remains in map index
	results = GetRegistry().GetInField(ten, f)
	assert.Len(t, results, 1)
	assert.Equal(t, reactor2.Id(), results[0].Id())
}

// TestRegistry_UniqueIds verifies ID generation produces unique IDs
func TestRegistry_UniqueIds(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()

	ids := make(map[uint32]bool)

	// Create multiple reactors and verify unique IDs
	for i := 0; i < 100; i++ {
		r := createTestReactor(ten, 1, 1, 100000, uint32(2000000+i), "reactor")
		assert.False(t, ids[r.Id()], "Duplicate ID generated: %d", r.Id())
		ids[r.Id()] = true
	}
}

// TestRegistry_TryClaimSpot verifies the spatial slot guard that prevents
// duplicate reactors at the same (classification, x, y) within a map instance.
func TestRegistry_TryClaimSpot(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	mk := NewMapKey(f)

	assert.True(t, GetRegistry().TryClaimSpot(ten, mk, 2001, 231, 253), "first claim should succeed")
	assert.False(t, GetRegistry().TryClaimSpot(ten, mk, 2001, 231, 253), "second claim on the same spot should be rejected")

	// A different position in the same map is independent.
	assert.True(t, GetRegistry().TryClaimSpot(ten, mk, 2001, 610, 254))

	// A different classification at the same coords is also independent.
	assert.True(t, GetRegistry().TryClaimSpot(ten, mk, 2002, 231, 253))

	// Releasing the original spot lets it be re-claimed (simulates respawn
	// after destroy + cooldown expiry).
	GetRegistry().ReleaseSpot(ten, mk, 2001, 231, 253)
	assert.True(t, GetRegistry().TryClaimSpot(ten, mk, 2001, 231, 253))
}

// newTestData returns a data.Model populated from the given state/timeout maps.
// Uses data.Extract so the model mirrors what production code consumes from
// atlas-data, including state.Model conversion.
func newTestData(t *testing.T, stateInfo map[int8][]state.RestModel, timeoutInfo map[int8]int32, timeoutNextStateInfo map[int8]int8) data.Model {
	t.Helper()
	if timeoutInfo == nil {
		timeoutInfo = map[int8]int32{}
	}
	if timeoutNextStateInfo == nil {
		timeoutNextStateInfo = map[int8]int8{}
	}
	m, err := data.Extract(data.RestModel{
		Name:                 "test",
		StateInfo:            stateInfo,
		TimeoutInfo:          timeoutInfo,
		TimeoutNextStateInfo: timeoutNextStateInfo,
	})
	if err != nil {
		t.Fatalf("data.Extract failed: %v", err)
	}
	return m
}

// TestHit_BreakableReactorDestroysOnTerminal verifies the fix for reactor 2001:
// a reactor with only type-0 events and no synthesized 999s must destroy and
// record cooldown on the terminal transition.
func TestHit_BreakableReactorDestroysOnTerminal(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// Shape mirrors what atlas-data now returns for reactor 2001:
	// state 0 -> 1 via type-0 event; state 1 has no events (terminal).
	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 0, NextState: 1, ActiveSkills: []uint32{}}},
		},
		nil, nil,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 2001, "reactor-2001").
		SetState(0).SetPosition(231, 253).SetDelay(5000).SetDirection(0).SetData(d)
	created, err := GetRegistry().Create(ten, builder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Hit may return a Kafka producer error in unit-test environments where
	// no broker is reachable; the registry mutations under test happen
	// before any producer call, so we tolerate that error here.
	_ = Hit(l)(ctx)(created.Id(), 0, 0)

	// After the hit: reactor must be gone from the registry.
	if _, err := GetRegistry().Get(ten, created.Id()); err == nil {
		t.Fatal("reactor should have been destroyed on terminal-state transition, but still exists")
	}

	// Cooldown must be recorded at its (classification,x,y) for the map.
	mk := NewMapKey(f)
	if !GetRegistry().IsOnCooldown(ten, mk, 2001, 231, 253) {
		t.Fatal("cooldown should have been recorded after destroy")
	}
}

// TestHit_ItemReactorPersistsAtTerminal verifies that a reactor whose matched
// hit event is type 100 is kept alive at the terminal state (moonflower-style).
func TestHit_ItemReactorPersistsAtTerminal(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// State 0 -> 1 via a type-100 event. State 1 has no events (terminal).
	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 100, NextState: 1, ActiveSkills: []uint32{}}},
		},
		nil, nil,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9108000, "moonflower").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, _ := GetRegistry().Create(ten, builder)

	// Producer error from missing broker is tolerated; semantic check is on
	// the registry state.
	_ = Hit(l)(ctx)(created.Id(), 0, 0)

	// Reactor should still exist at state 1.
	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("reactor should have been kept alive at terminal (type-100 event); got error: %v", err)
	}
	if got.State() != 1 {
		t.Fatalf("state = %d, want 1", got.State())
	}
}

// TestHit_SkillReactorPersistsAtTerminal verifies types 5/6/7 (GPQ skill-gated)
// also persist at terminal.
func TestHit_SkillReactorPersistsAtTerminal(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// State 0 -> 1 via a type-5 event (any skill matches since ActiveSkills
	// is empty in our test; in production types 5/6/7 carry activeSkillID —
	// we're only exercising the persist rule here).
	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 5, NextState: 1, ActiveSkills: []uint32{}}},
		},
		nil, nil,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 6109013, "gpq-skill-reactor").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, _ := GetRegistry().Create(ten, builder)

	// Producer error from missing broker is tolerated; semantic check is on
	// the registry state.
	_ = Hit(l)(ctx)(created.Id(), 0, 0)

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("skill reactor should persist at terminal; got error: %v", err)
	}
	if got.State() != 1 {
		t.Fatalf("state = %d, want 1", got.State())
	}
}

// TestRegistry_ClearAllSpotsForMap verifies the bulk release used when
// teardown wipes a map instance.
func TestRegistry_ClearAllSpotsForMap(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	mk := NewMapKey(f)

	GetRegistry().TryClaimSpot(ten, mk, 2001, 231, 253)
	GetRegistry().TryClaimSpot(ten, mk, 2001, 610, 254)
	GetRegistry().TryClaimSpot(ten, mk, 2001, 1529, 132)

	GetRegistry().ClearAllSpotsForMap(ten, mk)

	assert.True(t, GetRegistry().TryClaimSpot(ten, mk, 2001, 231, 253))
	assert.True(t, GetRegistry().TryClaimSpot(ten, mk, 2001, 610, 254))
	assert.True(t, GetRegistry().TryClaimSpot(ten, mk, 2001, 1529, 132))
}

// TestRegistry_ClearAllCooldownsForMap verifies the bulk cooldown wipe used
// when teardown clears a map instance (mirrors TestRegistry_ClearAllSpotsForMap).
func TestRegistry_ClearAllCooldownsForMap(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	mk := NewMapKey(f)

	// Record cooldowns with a very long delay so they are definitely still active.
	GetRegistry().RecordCooldown(ten, mk, 2001, 231, 253, 60000)
	GetRegistry().RecordCooldown(ten, mk, 2001, 610, 254, 60000)
	GetRegistry().RecordCooldown(ten, mk, 2002, 100, 200, 60000)

	// All three should be on cooldown before the wipe.
	assert.True(t, GetRegistry().IsOnCooldown(ten, mk, 2001, 231, 253))
	assert.True(t, GetRegistry().IsOnCooldown(ten, mk, 2001, 610, 254))
	assert.True(t, GetRegistry().IsOnCooldown(ten, mk, 2002, 100, 200))

	GetRegistry().ClearAllCooldownsForMap(ten, mk)

	// All three should be off cooldown after the wipe.
	assert.False(t, GetRegistry().IsOnCooldown(ten, mk, 2001, 231, 253))
	assert.False(t, GetRegistry().IsOnCooldown(ten, mk, 2001, 610, 254))
	assert.False(t, GetRegistry().IsOnCooldown(ten, mk, 2002, 100, 200))
}

// TestCooldown_ExpiresByTimestamp verifies that RecordCooldown stores an
// expiry timestamp and IsOnCooldown returns false once the delay has elapsed.
func TestCooldown_ExpiresByTimestamp(t *testing.T) {
	setupTestRegistry(t)
	te := setupTestTenant()
	mk := NewMapKey(field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000)).Build())
	GetRegistry().RecordCooldown(te, mk, 9101000, 100, 200, 50) // 50ms delay
	if !GetRegistry().IsOnCooldown(te, mk, 9101000, 100, 200) {
		t.Fatalf("expected on cooldown immediately after record")
	}
	time.Sleep(70 * time.Millisecond)
	if GetRegistry().IsOnCooldown(te, mk, 9101000, 100, 200) {
		t.Fatalf("expected cooldown expired after delay")
	}
}

// TestSpot_ClaimIsExclusivePerPosition verifies TryClaimSpot exclusivity and
// that ReleaseSpot re-enables claiming the same position.
func TestSpot_ClaimIsExclusivePerPosition(t *testing.T) {
	setupTestRegistry(t)
	te := setupTestTenant()
	mk := NewMapKey(field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000)).Build())
	if !GetRegistry().TryClaimSpot(te, mk, 9101000, 10, 20) {
		t.Fatalf("first claim should succeed")
	}
	if GetRegistry().TryClaimSpot(te, mk, 9101000, 10, 20) {
		t.Fatalf("second claim on same spot must fail")
	}
	GetRegistry().ReleaseSpot(te, mk, 9101000, 10, 20)
	if !GetRegistry().TryClaimSpot(te, mk, 9101000, 10, 20) {
		t.Fatalf("claim after release should succeed")
	}
}
