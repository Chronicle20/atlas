package instance

import (
	"sync"
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupInstanceTestRegistry(t *testing.T) *InstanceRegistry {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitInstanceRegistry(rc)
	return getInstanceRegistry()
}

func newTestRoute() RouteModel {
	route, _ := NewRouteBuilder("test-route").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(3).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()
	return route
}

func TestFindOrCreateInstance_CreatesNew(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)

	assert.NotEqual(t, uuid.Nil, inst.InstanceId())
	assert.Equal(t, route.Id(), inst.RouteId())
	assert.Equal(t, tenantId, inst.TenantId())
	assert.Equal(t, Boarding, inst.State())

	loaded, ok := reg.GetInstance(inst.InstanceId())
	assert.True(t, ok)
	assert.Equal(t, 0, loaded.CharacterCount())
	assert.True(t, loaded.BoardingUntil().After(now))
	assert.True(t, loaded.ArrivalAt().After(loaded.BoardingUntil()))
}

func TestFindOrCreateInstance_ReusesExisting(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst1 := reg.FindOrCreateInstance(tenantId, route, now)
	inst2 := reg.FindOrCreateInstance(tenantId, route, now)

	assert.Equal(t, inst1.InstanceId(), inst2.InstanceId())
}

func TestFindOrCreateInstance_NewWhenFull(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute() // capacity = 3
	now := time.Now()

	inst1 := reg.FindOrCreateInstance(tenantId, route, now)
	reg.AddCharacter(inst1.InstanceId(), CharacterEntry{CharacterId: 1})
	reg.AddCharacter(inst1.InstanceId(), CharacterEntry{CharacterId: 2})
	reg.AddCharacter(inst1.InstanceId(), CharacterEntry{CharacterId: 3})

	inst2 := reg.FindOrCreateInstance(tenantId, route, now)
	assert.NotEqual(t, inst1.InstanceId(), inst2.InstanceId())
}

func TestFindOrCreateInstance_NewWhenBoardingExpired(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst1 := reg.FindOrCreateInstance(tenantId, route, now)

	// Advance time past boarding window
	later := now.Add(15 * time.Second)
	inst2 := reg.FindOrCreateInstance(tenantId, route, later)

	assert.NotEqual(t, inst1.InstanceId(), inst2.InstanceId())
}

func TestAddCharacter(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)
	ok, count := reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: 42, WorldId: 0, ChannelId: 1})

	assert.True(t, ok)
	assert.Equal(t, 1, count)

	loaded, ok2 := reg.GetInstance(inst.InstanceId())
	assert.True(t, ok2)
	assert.Equal(t, 1, loaded.CharacterCount())
	assert.True(t, loaded.HasCharacter(42))
}

func TestAddCharacter_InvalidInstance(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	ok, count := reg.AddCharacter(uuid.New(), CharacterEntry{CharacterId: 42})
	assert.False(t, ok)
	assert.Equal(t, 0, count)
}

func TestRemoveCharacter(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)
	reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: 1})
	reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: 2})

	empty := reg.RemoveCharacter(inst.InstanceId(), 1)
	assert.False(t, empty)

	loaded, ok := reg.GetInstance(inst.InstanceId())
	assert.True(t, ok)
	assert.Equal(t, 1, loaded.CharacterCount())
	assert.False(t, loaded.HasCharacter(1))
	assert.True(t, loaded.HasCharacter(2))
}

func TestRemoveCharacter_LastCharacter(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)
	reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: 1})

	empty := reg.RemoveCharacter(inst.InstanceId(), 1)
	assert.True(t, empty)

	loaded, ok := reg.GetInstance(inst.InstanceId())
	assert.True(t, ok)
	assert.Equal(t, 0, loaded.CharacterCount())
}

func TestTransitionToInTransit(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)
	assert.Equal(t, Boarding, inst.State())

	ok := reg.TransitionToInTransit(inst.InstanceId())
	assert.True(t, ok)

	loaded, ok2 := reg.GetInstance(inst.InstanceId())
	assert.True(t, ok2)
	assert.Equal(t, InTransit, loaded.State())
}

func TestTransitionToInTransit_AlreadyInTransit(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)
	reg.TransitionToInTransit(inst.InstanceId())

	ok := reg.TransitionToInTransit(inst.InstanceId())
	assert.False(t, ok)
}

func TestReleaseInstance(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)
	instanceId := inst.InstanceId()

	reg.ReleaseInstance(instanceId)

	_, ok := reg.GetInstance(instanceId)
	assert.False(t, ok)
}

func TestGetExpiredBoarding(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)
	reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: 1})

	// Before expiration
	expired := reg.GetExpiredBoarding(now.Add(5 * time.Second))
	assert.Empty(t, expired)

	// After expiration
	expired = reg.GetExpiredBoarding(now.Add(15 * time.Second))
	assert.Len(t, expired, 1)
	assert.Equal(t, inst.InstanceId(), expired[0].InstanceId())
}

func TestGetExpiredTransit(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)
	reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: 1})
	reg.TransitionToInTransit(inst.InstanceId())

	// Before arrival
	expired := reg.GetExpiredTransit(now.Add(30 * time.Second))
	assert.Empty(t, expired)

	// After arrival (boarding 10s + travel 30s = 40s)
	expired = reg.GetExpiredTransit(now.Add(45 * time.Second))
	assert.Len(t, expired, 1)
}

func TestGetStuck(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	inst := reg.FindOrCreateInstance(tenantId, route, now)

	// Before max lifetime
	stuck := reg.GetStuck(now.Add(60*time.Second), route.MaxLifetime())
	assert.Empty(t, stuck)

	// After max lifetime (80s + 1s)
	stuck = reg.GetStuck(now.Add(81*time.Second), route.MaxLifetime())
	assert.Len(t, stuck, 1)
	assert.Equal(t, inst.InstanceId(), stuck[0].InstanceId())
}

func TestGetAllActive(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	reg.FindOrCreateInstance(tenantId, route, now)
	reg.FindOrCreateInstance(tenantId, route, now.Add(15*time.Second))

	active := reg.GetAllActive()
	assert.Len(t, active, 2)
}

func TestConcurrentAccess(t *testing.T) {
	reg := setupInstanceTestRegistry(t)
	tenantId := uuid.New()
	route := newTestRoute()
	now := time.Now()

	var wg sync.WaitGroup
	for i := uint32(0); i < 10; i++ {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()
			inst := reg.FindOrCreateInstance(tenantId, route, now)
			reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: id})
		}(i)
	}
	wg.Wait()

	active := reg.GetAllActive()
	totalChars := 0
	for _, inst := range active {
		totalChars += inst.CharacterCount()
	}
	assert.Equal(t, 10, totalChars)
}
