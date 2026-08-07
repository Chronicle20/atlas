package monster

import (
	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"
	"atlas-maps/monster"
	"context"
	"errors"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()

	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	os.Exit(m.Run())
}

func TestCooldownSpawnPoint_Creation(t *testing.T) {
	sp := monster2.SpawnPoint{
		Id:       1,
		Template: 100100,
		X:        100,
		Y:        200,
		Fh:       10,
		Team:     0,
	}

	now := time.Now()
	csp := &CooldownSpawnPoint{
		SpawnPoint:  sp,
		NextSpawnAt: now,
	}

	if csp.SpawnPoint.Id != sp.Id {
		t.Errorf("Expected Id %d, got %d", sp.Id, csp.SpawnPoint.Id)
	}

	if csp.SpawnPoint.Template != sp.Template {
		t.Errorf("Expected Template %d, got %d", sp.Template, csp.SpawnPoint.Template)
	}

	if !csp.NextSpawnAt.Equal(now) {
		t.Errorf("Expected NextSpawnAt %v, got %v", now, csp.NextSpawnAt)
	}
}

func TestCooldownFiltering(t *testing.T) {
	now := time.Now()

	spawnPoints := []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1}, NextSpawnAt: now.Add(-1 * time.Second)}, // Eligible
		{SpawnPoint: monster2.SpawnPoint{Id: 2}, NextSpawnAt: now.Add(3 * time.Second)},  // On cooldown
		{SpawnPoint: monster2.SpawnPoint{Id: 3}, NextSpawnAt: now},                       // Eligible (equal time)
		{SpawnPoint: monster2.SpawnPoint{Id: 4}, NextSpawnAt: now.Add(-5 * time.Second)}, // Eligible
	}

	var eligibleCount int
	var eligibleIds []uint32

	for _, sp := range spawnPoints {
		if sp.NextSpawnAt.Before(now) || sp.NextSpawnAt.Equal(now) {
			eligibleCount++
			eligibleIds = append(eligibleIds, sp.SpawnPoint.Id)
		}
	}

	if eligibleCount != 3 {
		t.Errorf("Expected 3 eligible spawn points, got %d", eligibleCount)
	}

	expectedIds := []uint32{1, 3, 4}
	if len(eligibleIds) != len(expectedIds) {
		t.Errorf("Expected %d eligible IDs, got %d", len(expectedIds), len(eligibleIds))
	}

	for i, id := range expectedIds {
		if eligibleIds[i] != id {
			t.Errorf("Expected eligible ID %d at index %d, got %d", id, i, eligibleIds[i])
		}
	}
}

func TestCooldownUpdate(t *testing.T) {
	now := time.Now()
	cooldownDuration := 5 * time.Second

	sp := &CooldownSpawnPoint{
		SpawnPoint:  monster2.SpawnPoint{Id: 1},
		NextSpawnAt: now,
	}

	if !sp.NextSpawnAt.Equal(now) {
		t.Errorf("Expected initial NextSpawnAt to be %v, got %v", now, sp.NextSpawnAt)
	}

	sp.NextSpawnAt = now.Add(cooldownDuration)

	expectedTime := now.Add(cooldownDuration)
	if !sp.NextSpawnAt.Equal(expectedTime) {
		t.Errorf("Expected NextSpawnAt to be %v, got %v", expectedTime, sp.NextSpawnAt)
	}

	if sp.NextSpawnAt.Before(now) || sp.NextSpawnAt.Equal(now) {
		t.Error("Spawn point should be on cooldown")
	}

	future := now.Add(cooldownDuration + time.Second)
	if !sp.NextSpawnAt.Before(future) {
		t.Error("Spawn point should be eligible after cooldown expires")
	}
}

func TestSpawnPointCooldownMechanism(t *testing.T) {
	now := time.Now()

	registry := make(map[character.MapKey][]*CooldownSpawnPoint)
	mutexes := make(map[character.MapKey]*sync.RWMutex)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Field: f,
	}

	registry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1}, NextSpawnAt: now.Add(-1 * time.Second)}, // Eligible
		{SpawnPoint: monster2.SpawnPoint{Id: 2}, NextSpawnAt: now.Add(3 * time.Second)},  // On cooldown
		{SpawnPoint: monster2.SpawnPoint{Id: 3}, NextSpawnAt: now},                       // Eligible
	}
	mutexes[mapKey] = &sync.RWMutex{}

	mutex := mutexes[mapKey]
	spawnPoints := registry[mapKey]

	mutex.RLock()
	var eligibleIndices []int
	for i, sp := range spawnPoints {
		if sp.NextSpawnAt.Before(now) || sp.NextSpawnAt.Equal(now) {
			eligibleIndices = append(eligibleIndices, i)
		}
	}
	mutex.RUnlock()

	if len(eligibleIndices) != 2 {
		t.Errorf("Expected 2 eligible spawn points, got %d", len(eligibleIndices))
	}

	if len(eligibleIndices) > 0 {
		idx := eligibleIndices[0]
		mutex.Lock()
		spawnPoints[idx].NextSpawnAt = now.Add(5 * time.Second)
		mutex.Unlock()

		if !spawnPoints[idx].NextSpawnAt.After(now) {
			t.Error("Cooldown was not properly updated")
		}
	}
}

func TestThreadSafety(t *testing.T) {
	registry := make(map[character.MapKey][]*CooldownSpawnPoint)
	mutexes := make(map[character.MapKey]*sync.RWMutex)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Field: f,
	}

	now := time.Now()
	registry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3}, NextSpawnAt: now},
	}
	mutexes[mapKey] = &sync.RWMutex{}

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				mutex := mutexes[mapKey]
				mutex.RLock()
				_ = len(registry[mapKey])
				mutex.RUnlock()
				time.Sleep(time.Microsecond)
			}
		}()
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				mutex := mutexes[mapKey]
				mutex.Lock()
				if len(registry[mapKey]) > id {
					registry[mapKey][id].NextSpawnAt = time.Now().Add(5 * time.Second)
				}
				mutex.Unlock()
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("Test timed out - possible deadlock")
	}
}

func TestMapKeyIsolation(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(1, 1, 100000001).Build()
	f3 := field.NewBuilder(1, 2, 100000000).Build()
	f4 := field.NewBuilder(2, 1, 100000000).Build()
	mapKey1 := character.MapKey{Tenant: te, Field: f1}
	mapKey2 := character.MapKey{Tenant: te, Field: f2}
	mapKey3 := character.MapKey{Tenant: te, Field: f3}
	mapKey4 := character.MapKey{Tenant: te, Field: f4}

	now := time.Now()

	_ = registry.SetSpawnPointsForMap(ctx, mapKey1, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1}, NextSpawnAt: now},
	})
	_ = registry.SetSpawnPointsForMap(ctx, mapKey2, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 2}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3}, NextSpawnAt: now},
	})
	_ = registry.SetSpawnPointsForMap(ctx, mapKey3, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 4}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 5}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 6}, NextSpawnAt: now},
	})
	_ = registry.SetSpawnPointsForMap(ctx, mapKey4, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 7}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 8}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 9}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 10}, NextSpawnAt: now},
	})

	spawnPoints1, exists1 := registry.GetSpawnPointsForMap(ctx, mapKey1)
	if !exists1 || len(spawnPoints1) != 1 {
		t.Errorf("MapKey1 should have 1 spawn point, got %d", len(spawnPoints1))
	}

	spawnPoints2, exists2 := registry.GetSpawnPointsForMap(ctx, mapKey2)
	if !exists2 || len(spawnPoints2) != 2 {
		t.Errorf("MapKey2 should have 2 spawn points, got %d", len(spawnPoints2))
	}

	spawnPoints3, exists3 := registry.GetSpawnPointsForMap(ctx, mapKey3)
	if !exists3 || len(spawnPoints3) != 3 {
		t.Errorf("MapKey3 should have 3 spawn points, got %d", len(spawnPoints3))
	}

	spawnPoints4, exists4 := registry.GetSpawnPointsForMap(ctx, mapKey4)
	if !exists4 || len(spawnPoints4) != 4 {
		t.Errorf("MapKey4 should have 4 spawn points, got %d", len(spawnPoints4))
	}

	// Reserve mapKey1's spawn point, stamping its cooldown.
	if _, err := registry.ReserveEligibleSpawnPoints(ctx, mapKey1, 1, defaultSpawnCooldown, 1); err != nil {
		t.Fatalf("ReserveEligibleSpawnPoints failed: %v", err)
	}

	// Verify other maps are unaffected
	spawnPoints2After, _ := registry.GetSpawnPointsForMap(ctx, mapKey2)
	for _, sp := range spawnPoints2After {
		if sp.NextSpawnAt.After(now.Add(time.Second)) {
			t.Error("MapKey2 spawn points should not be affected by MapKey1 modifications")
		}
	}

	spawnPoints3After, _ := registry.GetSpawnPointsForMap(ctx, mapKey3)
	for _, sp := range spawnPoints3After {
		if sp.NextSpawnAt.After(now.Add(time.Second)) {
			t.Error("MapKey3 spawn points should not be affected by MapKey1 modifications")
		}
	}

	spawnPoints4After, _ := registry.GetSpawnPointsForMap(ctx, mapKey4)
	for _, sp := range spawnPoints4After {
		if sp.NextSpawnAt.After(now.Add(time.Second)) {
			t.Error("MapKey4 spawn points should not be affected by MapKey1 modifications")
		}
	}
}

func TestCount(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	now := time.Now()
	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3}, NextSpawnAt: now},
	})

	n, err := registry.Count(ctx, mapKey)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 spawn points, got %d", n)
	}

	empty := character.MapKey{Tenant: te, Field: field.NewBuilder(2, 2, 999).Build()}
	n2, err := registry.Count(ctx, empty)
	if err != nil {
		t.Fatalf("Count failed for empty map: %v", err)
	}
	if n2 != 0 {
		t.Errorf("expected 0 spawn points for uninitialized map, got %d", n2)
	}
}

func TestReserveEligibleSpawnPoints_ReservesAndStamps(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	now := time.Now()
	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100, MobTime: 10}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2, Template: 100101, MobTime: 10}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3, Template: 100102, MobTime: 10}, NextSpawnAt: now},
	})

	// Reserve up to 2 of 3 eligible points; a reserved point is stamped on cooldown.
	reserved, err := registry.ReserveEligibleSpawnPoints(ctx, mapKey, 2, defaultSpawnCooldown, 12345)
	if err != nil {
		t.Fatalf("ReserveEligibleSpawnPoints failed: %v", err)
	}
	if len(reserved) != 2 {
		t.Fatalf("expected 2 reserved, got %d", len(reserved))
	}

	// Only 1 remains eligible; asking for 5 returns just that 1.
	reserved2, err := registry.ReserveEligibleSpawnPoints(ctx, mapKey, 5, defaultSpawnCooldown, 999)
	if err != nil {
		t.Fatalf("ReserveEligibleSpawnPoints failed: %v", err)
	}
	if len(reserved2) != 1 {
		t.Fatalf("expected 1 remaining eligible, got %d", len(reserved2))
	}

	// All now on cooldown; further reservation returns none.
	reserved3, err := registry.ReserveEligibleSpawnPoints(ctx, mapKey, 5, defaultSpawnCooldown, 42)
	if err != nil {
		t.Fatalf("ReserveEligibleSpawnPoints failed: %v", err)
	}
	if len(reserved3) != 0 {
		t.Errorf("expected 0 eligible after all reserved, got %d", len(reserved3))
	}
}

func TestReserveEligibleSpawnPoints_CooldownDuration(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	now := time.Now()
	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100, MobTime: 30}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2, Template: 100101, MobTime: 0}, NextSpawnAt: now},
	})

	reserved, err := registry.ReserveEligibleSpawnPoints(ctx, mapKey, 2, defaultSpawnCooldown, 7)
	if err != nil {
		t.Fatalf("ReserveEligibleSpawnPoints failed: %v", err)
	}
	if len(reserved) != 2 {
		t.Fatalf("expected 2 reserved, got %d", len(reserved))
	}

	sps, exists := registry.GetSpawnPointsForMap(ctx, mapKey)
	if !exists || len(sps) != 2 {
		t.Fatalf("expected 2 spawn points in registry, got %d", len(sps))
	}

	for _, sp := range sps {
		switch sp.Id {
		case 1: // MobTime > 0 -> MobTime seconds
			want := now.Add(30 * time.Second)
			if sp.NextSpawnAt.Before(want.Add(-time.Second)) || sp.NextSpawnAt.After(want.Add(time.Second)) {
				t.Errorf("point 1 (MobTime 30): expected cooldown near %v, got %v", want, sp.NextSpawnAt)
			}
		case 2: // MobTime <= 0 -> default cooldown
			want := now.Add(defaultSpawnCooldown)
			if sp.NextSpawnAt.Before(want.Add(-time.Second)) || sp.NextSpawnAt.After(want.Add(time.Second)) {
				t.Errorf("point 2 (MobTime 0): expected default cooldown near %v, got %v", want, sp.NextSpawnAt)
			}
		}
	}
}

func TestReserveEligibleSpawnPoints_NonPositiveLimit(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	now := time.Now()
	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100, MobTime: 10}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2, Template: 100101, MobTime: 10}, NextSpawnAt: now},
	})

	reserved, err := registry.ReserveEligibleSpawnPoints(ctx, mapKey, 0, defaultSpawnCooldown, 1)
	if err != nil {
		t.Fatalf("ReserveEligibleSpawnPoints failed: %v", err)
	}
	if len(reserved) != 0 {
		t.Errorf("expected 0 reserved for limit 0, got %d", len(reserved))
	}

	// A non-positive limit must not touch any cooldowns: both points still eligible.
	after, err := registry.ReserveEligibleSpawnPoints(ctx, mapKey, 2, defaultSpawnCooldown, 1)
	if err != nil {
		t.Fatalf("ReserveEligibleSpawnPoints failed: %v", err)
	}
	if len(after) != 2 {
		t.Errorf("expected both points still eligible after no-op reserve, got %d", len(after))
	}
}

// TestReserveEligibleSpawnPoints_ConcurrentNoDoubleReserve is the direct
// registry-level guarantee behind the SpawnMonsters over-spawn fix: no matter
// how many callers race, each spawn point is reserved at most once per cooldown
// window and the total reserved never exceeds the number of eligible points.
func TestReserveEligibleSpawnPoints_ConcurrentNoDoubleReserve(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	now := time.Now()
	const points = 5
	sps := make([]*CooldownSpawnPoint, points)
	for i := 0; i < points; i++ {
		sps[i] = &CooldownSpawnPoint{
			SpawnPoint:  monster2.SpawnPoint{Id: uint32(i + 1), Template: 100100, MobTime: 10},
			NextSpawnAt: now,
		}
	}
	_ = registry.SetSpawnPointsForMap(ctx, mapKey, sps)

	var mu sync.Mutex
	seen := make(map[uint32]int)
	total := 0

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			res, err := registry.ReserveEligibleSpawnPoints(ctx, mapKey, points, defaultSpawnCooldown, seed)
			if err != nil {
				return
			}
			mu.Lock()
			for _, r := range res {
				seen[r.Id]++
			}
			total += len(res)
			mu.Unlock()
		}(int64(i + 1))
	}
	wg.Wait()

	if total != points {
		t.Errorf("expected exactly %d total reservations across concurrent callers, got %d", points, total)
	}
	for id, cnt := range seen {
		if cnt != 1 {
			t.Errorf("spawn point %d reserved %d times (want exactly 1)", id, cnt)
		}
	}
}

func TestResetCooldown(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	now := time.Now()

	// Create spawn points: boss with MobTime=30, normal with MobTime=0
	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 9001, MobTime: 30}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2, Template: 9001, MobTime: 30}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3, Template: 1001, MobTime: 0}, NextSpawnAt: now},
	})

	// Reset cooldown for template 9001 (boss) - should set NextSpawnAt = now + 30s
	registry.ResetCooldown(ctx, mapKey, 9001)

	// Boss spawn points should now be on cooldown, normal should still be eligible.
	sps, exists := registry.GetSpawnPointsForMap(ctx, mapKey)
	if !exists || len(sps) != 3 {
		t.Fatalf("expected 3 spawn points, got %d", len(sps))
	}
	checkNow := time.Now()
	eligibleCount := 0
	var eligibleTemplate uint32
	for _, sp := range sps {
		if !sp.NextSpawnAt.After(checkNow) {
			eligibleCount++
			eligibleTemplate = sp.Template
		}
	}
	if eligibleCount != 1 {
		t.Errorf("Expected 1 eligible (normal monster only) after boss cooldown reset, got %d", eligibleCount)
	}
	if eligibleTemplate != 1001 {
		t.Errorf("Expected eligible spawn point to be template 1001, got %d", eligibleTemplate)
	}
}

func TestInitializeForMap(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	mockDP := &mockDataProcessor{
		mockSpawnPoints: []monster2.SpawnPoint{
			{Id: 1, Template: 100100, MobTime: 10, X: 100, Y: 200, Fh: 10},
			{Id: 2, Template: 100101, MobTime: 10, X: 150, Y: 230, Fh: 11},
		},
	}

	err := registry.InitializeForMap(ctx, mapKey, mockDP, logrus.New())
	if err != nil {
		t.Fatalf("InitializeForMap failed: %v", err)
	}

	spawnPoints, exists := registry.GetSpawnPointsForMap(ctx, mapKey)
	if !exists {
		t.Fatal("Registry should exist after initialization")
	}
	if len(spawnPoints) != 2 {
		t.Errorf("Expected 2 spawn points, got %d", len(spawnPoints))
	}

	// Idempotent - second call should not change anything
	err = registry.InitializeForMap(ctx, mapKey, mockDP, logrus.New())
	if err != nil {
		t.Fatalf("Second InitializeForMap failed: %v", err)
	}

	spawnPoints, _ = registry.GetSpawnPointsForMap(ctx, mapKey)
	if len(spawnPoints) != 2 {
		t.Errorf("Expected 2 spawn points after second init, got %d", len(spawnPoints))
	}
}

// Mock implementations for testing
type mockCharacterProcessor struct {
	charactersInMap map[character.MapKey][]uint32
}

func (m *mockCharacterProcessor) GetCharactersInMap(_ uuid.UUID, f field.Model) ([]uint32, error) {
	for storedMapKey, characters := range m.charactersInMap {
		if storedMapKey.Field.WorldId() == f.WorldId() && storedMapKey.Field.ChannelId() == f.ChannelId() && storedMapKey.Field.MapId() == f.MapId() && storedMapKey.Field.Instance() == f.Instance() {
			return characters, nil
		}
	}
	return []uint32{}, nil
}

func (m *mockCharacterProcessor) GetMapsWithCharacters() []character.MapKey {
	keys := make([]character.MapKey, 0, len(m.charactersInMap))
	for key := range m.charactersInMap {
		keys = append(keys, key)
	}
	return keys
}

func (m *mockCharacterProcessor) GetCharactersInMapAllInstances(_ uuid.UUID, _ world.Id, _ channel.Id, _ _map.Id) ([]uint32, error) {
	return nil, nil
}

func (m *mockCharacterProcessor) Enter(_ uuid.UUID, _ field.Model, _ uint32) {
}

func (m *mockCharacterProcessor) Exit(_ uuid.UUID, _ field.Model, _ uint32) {
}

func (m *mockCharacterProcessor) ExitAll(_ uint32) {
}

type mockMonsterProcessor struct {
	monstersInMap   map[character.MapKey]int
	countErr        error
	createdMonsters []MockCreatedMonster
	mu              sync.Mutex
}

type MockCreatedMonster struct {
	Field     field.Model
	MonsterId uint32
	X         int16
	Y         int16
	Fh        int16
	Team      int8
}

func (m *mockMonsterProcessor) CountInMap(_ uuid.UUID, f field.Model) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	for storedMapKey, count := range m.monstersInMap {
		if storedMapKey.Field.WorldId() == f.WorldId() && storedMapKey.Field.ChannelId() == f.ChannelId() && storedMapKey.Field.MapId() == f.MapId() && storedMapKey.Field.Instance() == f.Instance() {
			return count, nil
		}
	}
	return 0, nil
}

func (m *mockMonsterProcessor) CreateMonster(_ uuid.UUID, f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createdMonsters = append(m.createdMonsters, MockCreatedMonster{
		Field:     f,
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      team,
	})
}

func (m *mockMonsterProcessor) GetInMapRect(_ field.Model, _, _, _, _ int16, _ uint32, _ ...requests.Configurator) ([]monster.RestModel, error) {
	return nil, nil
}

func (m *mockMonsterProcessor) GetCreatedMonsters() []MockCreatedMonster {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]MockCreatedMonster, len(m.createdMonsters))
	copy(result, m.createdMonsters)
	return result
}

func (m *mockMonsterProcessor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createdMonsters = nil
}

// mockDataProcessor implements monster2.Processor interface for testing
type mockDataProcessor struct {
	mockSpawnPoints []monster2.SpawnPoint
}

func (m *mockDataProcessor) SpawnPointProvider(_ _map.Id) model.Provider[[]monster2.SpawnPoint] {
	return func() ([]monster2.SpawnPoint, error) {
		return m.mockSpawnPoints, nil
	}
}

func (m *mockDataProcessor) SpawnableSpawnPointProvider(_ _map.Id) model.Provider[[]monster2.SpawnPoint] {
	return func() ([]monster2.SpawnPoint, error) {
		return m.mockSpawnPoints, nil
	}
}

func (m *mockDataProcessor) GetSpawnPoints(_ _map.Id) ([]monster2.SpawnPoint, error) {
	return m.mockSpawnPoints, nil
}

func (m *mockDataProcessor) GetSpawnableSpawnPoints(_ _map.Id) ([]monster2.SpawnPoint, error) {
	return m.mockSpawnPoints, nil
}

func TestSpawnMonsters_CooldownValidation(t *testing.T) {
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	registry := GetRegistry()
	registry.Reset(ctx)

	mockCharProc := &mockCharacterProcessor{
		charactersInMap: make(map[character.MapKey][]uint32),
	}
	mockMonsterProc := &mockMonsterProcessor{
		monstersInMap: make(map[character.MapKey]int),
	}

	mockSpawnPoints := []monster2.SpawnPoint{
		{Id: 1, Template: 100100, MobTime: 10, X: 100, Y: 200, Fh: 10, Team: 0},
		{Id: 2, Template: 100101, MobTime: 10, X: 150, Y: 230, Fh: 11, Team: 0},
		{Id: 3, Template: 100102, MobTime: 10, X: 200, Y: 260, Fh: 12, Team: 0},
		{Id: 4, Template: 100103, MobTime: 10, X: 250, Y: 290, Fh: 13, Team: 0},
		{Id: 5, Template: 100104, MobTime: 10, X: 300, Y: 320, Fh: 14, Team: 0},
	}

	mockDataProc := &mockDataProcessor{
		mockSpawnPoints: mockSpawnPoints,
	}

	processor := &ProcessorImpl{
		l:   logrus.New(),
		ctx: tctx,
		t:   te,
		dp:  mockDataProc,
		cp:  mockCharProc,
		mp:  mockMonsterProc,
	}

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()

	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	mockCharProc.charactersInMap[mapKey] = []uint32{1001, 1002}
	mockMonsterProc.monstersInMap[mapKey] = 0

	characterCount := 2
	spawnPointCount := len(mockSpawnPoints)
	spawnRate := 0.70 + (0.05 * float64(characterCount))
	expectedSpawns := int(math.Ceil(spawnRate * float64(spawnPointCount)))

	transactionId := uuid.New()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SpawnMonsters panicked: %v", r)
		}
	}()

	spawnErr := processor.SpawnMonsters(transactionId, f)

	if spawnErr != nil {
		t.Errorf("SpawnMonsters should not return error, got: %v", spawnErr)
	}

	// Allow goroutines to complete
	time.Sleep(500 * time.Millisecond)

	registrySpawnPoints, exists := registry.GetSpawnPointsForMap(ctx, mapKey)
	if !exists {
		t.Fatalf("Spawn point registry should exist for map key after SpawnMonsters execution")
	}

	if len(registrySpawnPoints) != spawnPointCount {
		t.Errorf("Expected %d spawn points in registry, got %d", spawnPointCount, len(registrySpawnPoints))
	}

	// Verify cooldown updates
	now := time.Now()
	updatedCount := 0
	for _, csp := range registrySpawnPoints {
		if csp.NextSpawnAt.After(now.Add(7*time.Second)) && csp.NextSpawnAt.Before(now.Add(12*time.Second)) {
			updatedCount++
		}
	}

	if updatedCount != expectedSpawns {
		t.Errorf("Expected %d spawn points to have updated cooldowns, got %d", expectedSpawns, updatedCount)
	}

	createdMonsters := mockMonsterProc.GetCreatedMonsters()
	if len(createdMonsters) != expectedSpawns {
		t.Errorf("Expected %d monsters to be created, got %d", expectedSpawns, len(createdMonsters))
	}

	for _, m := range createdMonsters {
		if m.Field.WorldId() != worldId {
			t.Errorf("Created monster should have WorldId %d, got %d", worldId, m.Field.WorldId())
		}
		if m.Field.ChannelId() != channelId {
			t.Errorf("Created monster should have ChannelId %d, got %d", channelId, m.Field.ChannelId())
		}
		if m.Field.MapId() != mapId {
			t.Errorf("Created monster should have MapId %d, got %d", mapId, m.Field.MapId())
		}

		found := false
		for _, sp := range mockSpawnPoints {
			if m.MonsterId == sp.Template &&
				m.X == sp.X &&
				m.Y == sp.Y &&
				m.Fh == sp.Fh &&
				m.Team == sp.Team {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Created monster %+v does not match any spawn point", m)
		}
	}
}

// TestSpawnMonsters_ConcurrentDoesNotOverspawn reproduces the map-40000
// over-spawn bug: two triggers (character-enter + the periodic respawn task)
// can invoke SpawnMonsters concurrently for the same field. Because the
// monstersInMap count lags (CreateMonster is async) and spawn-point
// reservation was a non-atomic check-then-reserve, concurrent passes each
// observed all spawn points eligible and each spawned the full deficit,
// producing more live monsters than the map has spawn points.
//
// Invariant: across any burst of concurrent spawn passes, the number of
// monsters created within a single cooldown window must never exceed the
// number of spawn points (each point may fire at most once per cooldown).
func TestSpawnMonsters_ConcurrentDoesNotOverspawn(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	mockCharProc := &mockCharacterProcessor{
		charactersInMap: make(map[character.MapKey][]uint32),
	}
	mockMonsterProc := &mockMonsterProcessor{
		monstersInMap: make(map[character.MapKey]int),
	}

	// Mirror map 40000: 3 spawn points for the same template.
	mockSpawnPoints := []monster2.SpawnPoint{
		{Id: 1, Template: 9300018, MobTime: 1, X: 505, Y: 155, Fh: 19, Team: -1},
		{Id: 2, Template: 9300018, MobTime: 1, X: 322, Y: 155, Fh: 15, Team: -1},
		{Id: 3, Template: 9300018, MobTime: 1, X: 711, Y: 155, Fh: 5, Team: -1},
	}
	spawnPointCount := len(mockSpawnPoints)

	mockDataProc := &mockDataProcessor{mockSpawnPoints: mockSpawnPoints}

	processor := &ProcessorImpl{
		l:   logrus.New(),
		ctx: tctx,
		t:   te,
		dp:  mockDataProc,
		cp:  mockCharProc,
		mp:  mockMonsterProc,
	}

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}

	mockCharProc.charactersInMap[mapKey] = []uint32{1001}
	// Count stays 0 for the whole burst: models the real-world lag where
	// async CreateMonster results are not yet visible to CountInMap.
	mockMonsterProc.monstersInMap[mapKey] = 0

	// Fire many concurrent spawn passes for the SAME field, as the
	// enter-trigger and periodic task would.
	const passes = 12
	var wg sync.WaitGroup
	for i := 0; i < passes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = processor.SpawnMonsters(uuid.New(), f)
		}()
	}
	wg.Wait()

	// Allow async CreateMonster goroutines to complete.
	time.Sleep(500 * time.Millisecond)

	created := len(mockMonsterProc.GetCreatedMonsters())
	if created > spawnPointCount {
		t.Errorf("over-spawn: created %d monsters for a map with %d spawn points (want <= %d)",
			created, spawnPointCount, spawnPointCount)
	}
}

// TestSpawnMonsters_CountErrorSkipsSpawn verifies that a transient failure to
// count monsters already in the map skips the spawn pass rather than assuming
// zero. Assuming zero would spawn the full deficit and over-populate the map.
func TestSpawnMonsters_CountErrorSkipsSpawn(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	mockCharProc := &mockCharacterProcessor{
		charactersInMap: make(map[character.MapKey][]uint32),
	}
	mockMonsterProc := &mockMonsterProcessor{
		monstersInMap: make(map[character.MapKey]int),
		countErr:      errors.New("atlas-monsters unavailable"),
	}

	mockDataProc := &mockDataProcessor{
		mockSpawnPoints: []monster2.SpawnPoint{
			{Id: 1, Template: 100100, MobTime: 10, X: 100, Y: 200, Fh: 10, Team: 0},
			{Id: 2, Template: 100101, MobTime: 10, X: 150, Y: 230, Fh: 11, Team: 0},
			{Id: 3, Template: 100102, MobTime: 10, X: 200, Y: 260, Fh: 12, Team: 0},
		},
	}

	processor := &ProcessorImpl{
		l: logrus.New(), ctx: tctx, t: te,
		dp: mockDataProc, cp: mockCharProc, mp: mockMonsterProc,
	}

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}
	mockCharProc.charactersInMap[mapKey] = []uint32{1001}

	err := processor.SpawnMonsters(uuid.New(), f)
	if err == nil {
		t.Error("expected SpawnMonsters to return the count error, got nil")
	}

	time.Sleep(100 * time.Millisecond)

	if created := len(mockMonsterProc.GetCreatedMonsters()); created != 0 {
		t.Errorf("expected 0 monsters created when count fails, got %d", created)
	}
}

func TestSpawnMonsters_NoCharacters(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	mockCharProc := &mockCharacterProcessor{
		charactersInMap: make(map[character.MapKey][]uint32),
	}
	mockMonsterProc := &mockMonsterProcessor{
		monstersInMap: make(map[character.MapKey]int),
	}

	mockSpawnPoint := monster2.SpawnPoint{
		Id: 1, Template: 100100, MobTime: 10, X: 100, Y: 200, Fh: 10, Team: 0,
	}

	mockDataProc := &mockDataProcessor{
		mockSpawnPoints: []monster2.SpawnPoint{mockSpawnPoint},
	}

	processor := &ProcessorImpl{
		l:   logrus.New(),
		ctx: tctx,
		t:   te,
		dp:  mockDataProc,
		cp:  mockCharProc,
		mp:  mockMonsterProc,
	}

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()

	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	mockCharProc.charactersInMap[mapKey] = []uint32{}
	mockMonsterProc.monstersInMap[mapKey] = 0
	mockMonsterProc.Reset()

	transactionId := uuid.New()
	err := processor.SpawnMonsters(transactionId, f)
	if err != nil {
		t.Errorf("SpawnMonsters should not return error, got: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	createdMonsters := mockMonsterProc.GetCreatedMonsters()
	if len(createdMonsters) != 0 {
		t.Errorf("Expected 0 monsters to be created when no characters present, got %d", len(createdMonsters))
	}

	registrySpawnPoints, exists := registry.GetSpawnPointsForMap(ctx, mapKey)
	if !exists {
		t.Error("Registry should be initialized even when no characters present")
	}

	if len(registrySpawnPoints) != 1 {
		t.Errorf("Expected 1 spawn point in registry, got %d", len(registrySpawnPoints))
	}
}

func TestSpawnMonsters_AllSpawnPointsOnCooldown(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	mockCharProc := &mockCharacterProcessor{
		charactersInMap: make(map[character.MapKey][]uint32),
	}
	mockMonsterProc := &mockMonsterProcessor{
		monstersInMap: make(map[character.MapKey]int),
	}

	mockSpawnPoints := []monster2.SpawnPoint{
		{Id: 1, Template: 100100, MobTime: 10, X: 100, Y: 200, Fh: 10, Team: 0},
		{Id: 2, Template: 100101, MobTime: 10, X: 150, Y: 200, Fh: 10, Team: 0},
	}

	mockDataProc := &mockDataProcessor{
		mockSpawnPoints: mockSpawnPoints,
	}

	processor := &ProcessorImpl{
		l:   logrus.New(),
		ctx: tctx,
		t:   te,
		dp:  mockDataProc,
		cp:  mockCharProc,
		mp:  mockMonsterProc,
	}

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()

	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	mockCharProc.charactersInMap[mapKey] = []uint32{1001, 1002}
	mockMonsterProc.monstersInMap[mapKey] = 0
	mockMonsterProc.Reset()

	// Pre-initialize with all spawn points on cooldown
	futureTime := time.Now().Add(5 * time.Second)
	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: mockSpawnPoints[0], NextSpawnAt: futureTime},
		{SpawnPoint: mockSpawnPoints[1], NextSpawnAt: futureTime},
	})

	transactionId := uuid.New()
	err := processor.SpawnMonsters(transactionId, f)
	if err != nil {
		t.Errorf("SpawnMonsters should not return error, got: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	createdMonsters := mockMonsterProc.GetCreatedMonsters()
	if len(createdMonsters) != 0 {
		t.Errorf("Expected 0 monsters to be created when all spawn points on cooldown, got %d", len(createdMonsters))
	}
}
