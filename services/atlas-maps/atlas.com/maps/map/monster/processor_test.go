package monster

import (
	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"
	"context"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
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

func TestProcessorImpl_shuffleIndices(t *testing.T) {
	processor := &ProcessorImpl{}

	indices := []int{0, 1, 2, 3, 4}
	shuffled := processor.shuffleIndices(indices)

	if len(shuffled) != len(indices) {
		t.Errorf("Expected shuffled length %d, got %d", len(indices), len(shuffled))
	}

	for _, idx := range shuffled {
		if idx < 0 || idx >= len(indices) {
			t.Errorf("Invalid index %d in shuffled result", idx)
		}
	}

	results := make([][]int, 10)
	for i := 0; i < 10; i++ {
		results[i] = processor.shuffleIndices(indices)
	}

	allSame := true
	for i := 1; i < len(results); i++ {
		if !sliceEqual(results[0], results[i]) {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("All shuffle results are identical - randomization may not be working")
	}
}

func TestProcessorImpl_shuffle(t *testing.T) {
	processor := &ProcessorImpl{}

	originalSpawnPoints := []monster2.SpawnPoint{
		{Id: 1, Template: 100},
		{Id: 2, Template: 200},
		{Id: 3, Template: 300},
		{Id: 4, Template: 400},
	}

	shuffled := processor.shuffle(originalSpawnPoints)

	if len(shuffled) != len(originalSpawnPoints) {
		t.Errorf("Expected shuffled length %d, got %d", len(originalSpawnPoints), len(shuffled))
	}

	for _, original := range originalSpawnPoints {
		found := false
		for _, shuffledSp := range shuffled {
			if shuffledSp.Id == original.Id && shuffledSp.Template == original.Template {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Original spawn point with Id %d not found in shuffled result", original.Id)
		}
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

func TestConcurrentSpawningAcrossMultipleMaps(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(1, 1, 100000001).Build()
	f3 := field.NewBuilder(1, 2, 100000000).Build()
	f4 := field.NewBuilder(2, 1, 100000000).Build()
	f5 := field.NewBuilder(1, 1, 100000002).Build()
	mapKeys := []character.MapKey{
		{Field: f1}, {Field: f2}, {Field: f3}, {Field: f4}, {Field: f5},
	}

	now := time.Now()

	for i, mapKey := range mapKeys {
		spawnPoints := []*CooldownSpawnPoint{
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 1)}, NextSpawnAt: now},
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 2)}, NextSpawnAt: now},
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 3)}, NextSpawnAt: now},
		}
		if err := registry.SetSpawnPointsForMap(ctx, mapKey, spawnPoints); err != nil {
			t.Fatalf("Failed to set spawn points: %v", err)
		}
	}

	var wg sync.WaitGroup

	for _, mapKey := range mapKeys {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(mk character.MapKey) {
				defer wg.Done()
				_, _, _ = registry.GetEligibleSpawnPoints(ctx, mk)
			}(mapKey)
		}
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Error("Test timed out - possible deadlock in concurrent spawning")
	}

	for _, mapKey := range mapKeys {
		spawnPoints, exists := registry.GetSpawnPointsForMap(ctx, mapKey)
		if !exists {
			t.Errorf("Registry for map %s should exist", mapKeyToString(mapKey))
		}
		if len(spawnPoints) != 3 {
			t.Errorf("Expected 3 spawn points, got %d", len(spawnPoints))
		}
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

	// Update mapKey1 cooldown
	updates := map[uint32]time.Time{spawnPoints1[0].SpawnPoint.Id: now.Add(10 * time.Second)}
	_ = registry.UpdateCooldowns(ctx, mapKey1, updates)

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

func TestMultiMapConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	registry := GetRegistry()
	registry.Reset(ctx)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	numMaps := 5
	mapKeys := make([]character.MapKey, numMaps)
	for i := 0; i < numMaps; i++ {
		f := field.NewBuilder(1, 1, _map.Id(100000000+i)).Build()
		mapKeys[i] = character.MapKey{
			Tenant: te,
			Field:  f,
		}
	}

	now := time.Now()

	for i, mapKey := range mapKeys {
		_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 1)}, NextSpawnAt: now},
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 2)}, NextSpawnAt: now},
		})
	}

	var wg sync.WaitGroup
	const iterations = 50

	for _, mapKey := range mapKeys {
		wg.Add(1)
		go func(mk character.MapKey) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				// Read operations
				_, _, _ = registry.GetEligibleSpawnPoints(ctx, mk)

				// Write operations
				if i%5 == 0 {
					updates := map[uint32]time.Time{1: time.Now().Add(5 * time.Second)}
					_ = registry.UpdateCooldowns(ctx, mk, updates)
				}

				time.Sleep(time.Microsecond)
			}
		}(mapKey)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Error("Test timed out - possible deadlock in multi-map concurrent access")
	}

	for _, mapKey := range mapKeys {
		spawnPoints, exists := registry.GetSpawnPointsForMap(ctx, mapKey)
		if !exists || len(spawnPoints) != 2 {
			t.Errorf("Map should have 2 spawn points after concurrent access, got %d", len(spawnPoints))
		}
	}
}

func TestCooldownEnforcementPreventsImmediateRespawn(t *testing.T) {
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

	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2, Template: 100101}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3, Template: 100102}, NextSpawnAt: now},
	})

	// All 3 should be eligible initially
	eligible, total, err := registry.GetEligibleSpawnPoints(ctx, mapKey)
	if err != nil {
		t.Fatalf("GetEligibleSpawnPoints failed: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected 3 total spawn points, got %d", total)
	}
	if len(eligible) != 3 {
		t.Errorf("Expected 3 eligible spawn points initially, got %d", len(eligible))
	}

	// Simulate spawning from first spawn point (update cooldown)
	updates := map[uint32]time.Time{eligible[0].SpawnPoint.Id: now.Add(5 * time.Second)}
	_ = registry.UpdateCooldowns(ctx, mapKey, updates)

	// Should have 2 eligible after one is on cooldown
	eligibleAfter, _, err := registry.GetEligibleSpawnPoints(ctx, mapKey)
	if err != nil {
		t.Fatalf("GetEligibleSpawnPoints failed: %v", err)
	}
	if len(eligibleAfter) != 2 {
		t.Errorf("Expected 2 eligible spawn points after first spawn, got %d", len(eligibleAfter))
	}

	// Put all remaining on cooldown
	allUpdates := make(map[uint32]time.Time)
	for _, csp := range eligibleAfter {
		allUpdates[csp.SpawnPoint.Id] = now.Add(5 * time.Second)
	}
	_ = registry.UpdateCooldowns(ctx, mapKey, allUpdates)

	// Should have 0 eligible
	eligibleFinal, _, err := registry.GetEligibleSpawnPoints(ctx, mapKey)
	if err != nil {
		t.Fatalf("GetEligibleSpawnPoints failed: %v", err)
	}
	if len(eligibleFinal) != 0 {
		t.Errorf("Expected 0 eligible spawn points after all spawns, got %d", len(eligibleFinal))
	}
}

func TestCooldownTimingAccuracy(t *testing.T) {
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

	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: now},
	})

	// Set cooldown to 5 seconds from now
	cooldownTime := now.Add(5 * time.Second)
	_ = registry.UpdateCooldowns(ctx, mapKey, map[uint32]time.Time{1: cooldownTime})

	// Verify spawn point is on cooldown (not eligible now)
	eligible, _, err := registry.GetEligibleSpawnPoints(ctx, mapKey)
	if err != nil {
		t.Fatalf("GetEligibleSpawnPoints failed: %v", err)
	}
	if len(eligible) != 0 {
		t.Error("Spawn point should not be eligible while on cooldown")
	}

	// Verify the stored cooldown time
	spawnPoints, exists := registry.GetSpawnPointsForMap(ctx, mapKey)
	if !exists || len(spawnPoints) != 1 {
		t.Fatal("Expected 1 spawn point in registry")
	}

	// NextSpawnAt should be approximately cooldownTime (within 1 second tolerance for serialization)
	if spawnPoints[0].NextSpawnAt.Before(cooldownTime.Add(-time.Second)) || spawnPoints[0].NextSpawnAt.After(cooldownTime.Add(time.Second)) {
		t.Errorf("Expected NextSpawnAt near %v, got %v", cooldownTime, spawnPoints[0].NextSpawnAt)
	}
}

func TestSequentialSpawnAttempts(t *testing.T) {
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

	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2, Template: 100101}, NextSpawnAt: now},
	})

	// Both eligible initially
	eligible, _, _ := registry.GetEligibleSpawnPoints(ctx, mapKey)
	if len(eligible) != 2 {
		t.Errorf("Expected 2 eligible spawn points initially, got %d", len(eligible))
	}

	// Simulate spawning from first spawn point
	_ = registry.UpdateCooldowns(ctx, mapKey, map[uint32]time.Time{1: now.Add(5 * time.Second)})

	// One eligible
	eligible, _, _ = registry.GetEligibleSpawnPoints(ctx, mapKey)
	if len(eligible) != 1 {
		t.Errorf("Expected 1 eligible spawn point after first spawn, got %d", len(eligible))
	}

	// Simulate spawning from second spawn point
	_ = registry.UpdateCooldowns(ctx, mapKey, map[uint32]time.Time{2: now.Add(5 * time.Second)})

	// None eligible
	eligible, _, _ = registry.GetEligibleSpawnPoints(ctx, mapKey)
	if len(eligible) != 0 {
		t.Errorf("Expected 0 eligible spawn points after all spawns, got %d", len(eligible))
	}
}

func TestRapidSpawnAttempts(t *testing.T) {
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

	_ = registry.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: now},
	})

	// First check - should be eligible
	eligible, _, _ := registry.GetEligibleSpawnPoints(ctx, mapKey)
	if len(eligible) != 1 {
		t.Error("First spawn should be eligible")
	}

	// Simulate spawn and set cooldown
	_ = registry.UpdateCooldowns(ctx, mapKey, map[uint32]time.Time{1: now.Add(5 * time.Second)})

	// Rapid checks should all show not eligible
	for i := 0; i < 10; i++ {
		eligible, _, _ = registry.GetEligibleSpawnPoints(ctx, mapKey)
		if len(eligible) != 0 {
			t.Errorf("Rapid spawn attempt %d should be blocked by cooldown", i)
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

	// All eligible initially
	eligible, _, _ := registry.GetEligibleSpawnPoints(ctx, mapKey)
	if len(eligible) != 3 {
		t.Errorf("Expected 3 eligible, got %d", len(eligible))
	}

	// Reset cooldown for template 9001 (boss) - should set NextSpawnAt = now + 30s
	registry.ResetCooldown(ctx, mapKey, 9001)

	// Boss spawn points should now be on cooldown, normal should still be eligible
	eligible, _, _ = registry.GetEligibleSpawnPoints(ctx, mapKey)
	if len(eligible) != 1 {
		t.Errorf("Expected 1 eligible (normal monster only) after boss cooldown reset, got %d", len(eligible))
	}
	if eligible[0].SpawnPoint.Template != 1001 {
		t.Errorf("Expected eligible spawn point to be template 1001, got %d", eligible[0].SpawnPoint.Template)
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

// Helper function to convert MapKey to string for testing
func mapKeyToString(mk character.MapKey) string {
	return string(mk.Field.Id())
}

// Helper function to compare slices
func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

type mockMonsterProcessor struct {
	monstersInMap   map[character.MapKey]int
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
