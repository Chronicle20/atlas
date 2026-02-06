package monster

import (
	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

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

	// Test that the cooldown spawn point was created correctly
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

	// Check that the correct spawn points are eligible
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

	// Test initial state - should be eligible
	if !sp.NextSpawnAt.Equal(now) {
		t.Errorf("Expected initial NextSpawnAt to be %v, got %v", now, sp.NextSpawnAt)
	}

	// Simulate spawn and cooldown update
	sp.NextSpawnAt = now.Add(cooldownDuration)

	// Check that cooldown is properly set
	expectedTime := now.Add(cooldownDuration)
	if !sp.NextSpawnAt.Equal(expectedTime) {
		t.Errorf("Expected NextSpawnAt to be %v, got %v", expectedTime, sp.NextSpawnAt)
	}

	// Check that spawn point is on cooldown
	if sp.NextSpawnAt.Before(now) || sp.NextSpawnAt.Equal(now) {
		t.Error("Spawn point should be on cooldown")
	}

	// Test that after cooldown expires, spawn point becomes eligible again
	future := now.Add(cooldownDuration + time.Second)
	if !sp.NextSpawnAt.Before(future) {
		t.Error("Spawn point should be eligible after cooldown expires")
	}
}

func TestProcessorImpl_shuffleIndices(t *testing.T) {
	processor := &ProcessorImpl{}

	indices := []int{0, 1, 2, 3, 4}
	shuffled := processor.shuffleIndices(indices)

	// Check that length is preserved
	if len(shuffled) != len(indices) {
		t.Errorf("Expected shuffled length %d, got %d", len(indices), len(shuffled))
	}

	// Check that all indices are within expected range
	for _, idx := range shuffled {
		if idx < 0 || idx >= len(indices) {
			t.Errorf("Invalid index %d in shuffled result", idx)
		}
	}

	// Run shuffle multiple times to verify it produces different results
	results := make([][]int, 10)
	for i := 0; i < 10; i++ {
		results[i] = processor.shuffleIndices(indices)
	}

	// Check that at least one result is different (very high probability)
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

	// Check that length is preserved
	if len(shuffled) != len(originalSpawnPoints) {
		t.Errorf("Expected shuffled length %d, got %d", len(originalSpawnPoints), len(shuffled))
	}

	// Check that all original spawn points are present
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

	// Create a registry with spawn points
	registry := make(map[character.MapKey][]*CooldownSpawnPoint)
	mutexes := make(map[character.MapKey]*sync.RWMutex)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Field: f,
	}

	// Initialize spawn points
	registry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1}, NextSpawnAt: now.Add(-1 * time.Second)}, // Eligible
		{SpawnPoint: monster2.SpawnPoint{Id: 2}, NextSpawnAt: now.Add(3 * time.Second)},  // On cooldown
		{SpawnPoint: monster2.SpawnPoint{Id: 3}, NextSpawnAt: now},                       // Eligible
	}
	mutexes[mapKey] = &sync.RWMutex{}

	// Test filtering eligible spawn points
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

	// Test updating cooldown after spawn
	if len(eligibleIndices) > 0 {
		idx := eligibleIndices[0]
		mutex.Lock()
		spawnPoints[idx].NextSpawnAt = now.Add(5 * time.Second)
		mutex.Unlock()

		// Verify cooldown was updated
		if !spawnPoints[idx].NextSpawnAt.After(now) {
			t.Error("Cooldown was not properly updated")
		}
	}
}

func TestThreadSafety(t *testing.T) {
	// Create a registry with spawn points
	registry := make(map[character.MapKey][]*CooldownSpawnPoint)
	mutexes := make(map[character.MapKey]*sync.RWMutex)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Field: f,
	}

	// Initialize spawn points
	now := time.Now()
	registry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3}, NextSpawnAt: now},
	}
	mutexes[mapKey] = &sync.RWMutex{}

	// Test concurrent access
	var wg sync.WaitGroup
	iterations := 100

	// Readers
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

	// Writers
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

	// Wait for all goroutines to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(5 * time.Second):
		t.Error("Test timed out - possible deadlock")
	}
}

func TestConcurrentSpawningAcrossMultipleMaps(t *testing.T) {
	// Test concurrent spawning across multiple maps to ensure proper isolation
	registry := GetRegistry()
	registry.Reset()

	// Create multiple map keys for different maps
	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(1, 1, 100000001).Build()
	f3 := field.NewBuilder(1, 2, 100000000).Build()
	f4 := field.NewBuilder(2, 1, 100000000).Build()
	f5 := field.NewBuilder(1, 1, 100000002).Build()
	mapKeys := []character.MapKey{
		{Field: f1},
		{Field: f2},
		{Field: f3},
		{Field: f4},
		{Field: f5},
	}

	now := time.Now()

	// Initialize registry for each map using the singleton registry
	for i, mapKey := range mapKeys {
		spawnPoints := []*CooldownSpawnPoint{
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 1)}, NextSpawnAt: now},
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 2)}, NextSpawnAt: now},
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 3)}, NextSpawnAt: now},
		}

		// Manually initialize the registry for this map
		registry.registryMu.Lock()
		registry.spawnPointRegistry[mapKey] = spawnPoints
		registry.spawnPointMu[mapKey] = &sync.RWMutex{}
		registry.registryMu.Unlock()
	}

	// Track operations for verification
	var operationCount sync.Map
	var wg sync.WaitGroup

	// Simulate concurrent spawning operations on different maps
	for _, mapKey := range mapKeys {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(mk character.MapKey, iteration int) {
				defer wg.Done()

				spawnPoints, mutex, err := registry.GetOrInitializeSpawnPoints(mk, nil, logrus.New())
				if err != nil {
					t.Errorf("Failed to get spawn points: %v", err)
					return
				}

				// Simulate spawn operation
				mutex.RLock()
				var eligibleIndices []int
				for idx, sp := range spawnPoints {
					if sp.NextSpawnAt.Before(time.Now()) || sp.NextSpawnAt.Equal(sp.NextSpawnAt) {
						eligibleIndices = append(eligibleIndices, idx)
					}
				}
				mutex.RUnlock()

				// Update cooldown if spawn points are available
				if len(eligibleIndices) > 0 {
					selectedIdx := eligibleIndices[iteration%len(eligibleIndices)]
					mutex.Lock()
					spawnPoints[selectedIdx].NextSpawnAt = time.Now().Add(5 * time.Second)
					mutex.Unlock()

					// Track operation
					key := mapKeyToString(mk)
					if count, ok := operationCount.Load(key); ok {
						operationCount.Store(key, count.(int)+1)
					} else {
						operationCount.Store(key, 1)
					}
				}
			}(mapKey, i)
		}
	}

	// Wait for all operations to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(10 * time.Second):
		t.Error("Test timed out - possible deadlock in concurrent spawning")
	}

	// Verify that operations were performed on all maps
	operationCount.Range(func(key, value interface{}) bool {
		if value.(int) == 0 {
			t.Errorf("No operations performed on map %s", key.(string))
		}
		return true
	})

	// Verify registry isolation - each map should have its own registry
	for _, mapKey := range mapKeys {
		if _, exists := registry.GetSpawnPointsForMap(mapKey); !exists {
			t.Errorf("Registry for map %s should exist", mapKeyToString(mapKey))
		}
	}

	// Verify that different maps have different registry entries
	registry.registryMu.RLock()
	registryCount := len(registry.spawnPointRegistry)
	mutexCount := len(registry.spawnPointMu)
	registry.registryMu.RUnlock()

	if registryCount != len(mapKeys) {
		t.Errorf("Expected %d registry entries, got %d", len(mapKeys), registryCount)
	}

	if mutexCount != len(mapKeys) {
		t.Errorf("Expected %d mutex entries, got %d", len(mapKeys), mutexCount)
	}
}

func TestMapKeyIsolation(t *testing.T) {
	// Test that different MapKeys maintain separate registries
	registry := GetRegistry()
	registry.Reset()

	// Create test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	// Create different map keys
	f1 := field.NewBuilder(1, 1, 100000000).Build()
	f2 := field.NewBuilder(1, 1, 100000001).Build()
	f3 := field.NewBuilder(1, 2, 100000000).Build()
	f4 := field.NewBuilder(2, 1, 100000000).Build()
	mapKey1 := character.MapKey{Tenant: te, Field: f1}
	mapKey2 := character.MapKey{Tenant: te, Field: f2}
	mapKey3 := character.MapKey{Tenant: te, Field: f3}
	mapKey4 := character.MapKey{Tenant: te, Field: f4}

	now := time.Now()

	// Initialize registries with different spawn points using direct access
	registry.registryMu.Lock()
	registry.spawnPointRegistry[mapKey1] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1}, NextSpawnAt: now},
	}
	registry.spawnPointMu[mapKey1] = &sync.RWMutex{}

	registry.spawnPointRegistry[mapKey2] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 2}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3}, NextSpawnAt: now},
	}
	registry.spawnPointMu[mapKey2] = &sync.RWMutex{}

	registry.spawnPointRegistry[mapKey3] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 4}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 5}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 6}, NextSpawnAt: now},
	}
	registry.spawnPointMu[mapKey3] = &sync.RWMutex{}

	registry.spawnPointRegistry[mapKey4] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 7}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 8}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 9}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 10}, NextSpawnAt: now},
	}
	registry.spawnPointMu[mapKey4] = &sync.RWMutex{}
	registry.registryMu.Unlock()

	// Verify that each map has the correct number of spawn points
	spawnPoints1, exists1 := registry.GetSpawnPointsForMap(mapKey1)
	if !exists1 || len(spawnPoints1) != 1 {
		t.Errorf("MapKey1 should have 1 spawn point, got %d", len(spawnPoints1))
	}

	spawnPoints2, exists2 := registry.GetSpawnPointsForMap(mapKey2)
	if !exists2 || len(spawnPoints2) != 2 {
		t.Errorf("MapKey2 should have 2 spawn points, got %d", len(spawnPoints2))
	}

	spawnPoints3, exists3 := registry.GetSpawnPointsForMap(mapKey3)
	if !exists3 || len(spawnPoints3) != 3 {
		t.Errorf("MapKey3 should have 3 spawn points, got %d", len(spawnPoints3))
	}

	spawnPoints4, exists4 := registry.GetSpawnPointsForMap(mapKey4)
	if !exists4 || len(spawnPoints4) != 4 {
		t.Errorf("MapKey4 should have 4 spawn points, got %d", len(spawnPoints4))
	}

	// Verify that spawn points are isolated (different IDs)
	if spawnPoints1[0].SpawnPoint.Id != 1 {
		t.Errorf("MapKey1 spawn point should have ID 1, got %d", spawnPoints1[0].SpawnPoint.Id)
	}

	if spawnPoints2[0].SpawnPoint.Id != 2 {
		t.Errorf("MapKey2 first spawn point should have ID 2, got %d", spawnPoints2[0].SpawnPoint.Id)
	}

	// Verify that modifying one map doesn't affect others
	registry.registryMu.RLock()
	mutex1 := registry.spawnPointMu[mapKey1]
	registry.registryMu.RUnlock()

	mutex1.Lock()
	spawnPoints1[0].NextSpawnAt = now.Add(10 * time.Second)
	mutex1.Unlock()

	// Check that other maps are unaffected
	spawnPoints2After, _ := registry.GetSpawnPointsForMap(mapKey2)
	if spawnPoints2After[0].NextSpawnAt.After(now.Add(time.Second)) {
		t.Error("MapKey2 spawn points should not be affected by MapKey1 modifications")
	}

	spawnPoints3After, _ := registry.GetSpawnPointsForMap(mapKey3)
	if spawnPoints3After[0].NextSpawnAt.After(now.Add(time.Second)) {
		t.Error("MapKey3 spawn points should not be affected by MapKey1 modifications")
	}

	spawnPoints4After, _ := registry.GetSpawnPointsForMap(mapKey4)
	if spawnPoints4After[0].NextSpawnAt.After(now.Add(time.Second)) {
		t.Error("MapKey4 spawn points should not be affected by MapKey1 modifications")
	}
}

func TestMultiMapConcurrentAccess(t *testing.T) {
	// Test that concurrent access to different maps doesn't interfere
	registry := GetRegistry()
	registry.Reset()

	// Create test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	// Create multiple maps
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

	// Initialize all maps using direct registry access
	registry.registryMu.Lock()
	for i, mapKey := range mapKeys {
		registry.spawnPointRegistry[mapKey] = []*CooldownSpawnPoint{
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 1)}, NextSpawnAt: now},
			{SpawnPoint: monster2.SpawnPoint{Id: uint32(i*10 + 2)}, NextSpawnAt: now},
		}
		registry.spawnPointMu[mapKey] = &sync.RWMutex{}
	}
	registry.registryMu.Unlock()

	var wg sync.WaitGroup
	const iterations = 50

	// Start concurrent operations on each map
	for mapIndex, mapKey := range mapKeys {
		wg.Add(1)
		go func(mk character.MapKey, index int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				// Get spawn points and mutex safely
				spawnPoints, exists := registry.GetSpawnPointsForMap(mk)
				if !exists || len(spawnPoints) == 0 {
					continue
				}

				registry.registryMu.RLock()
				mutex := registry.spawnPointMu[mk]
				registry.registryMu.RUnlock()

				// Read operations (multiple concurrent readers)
				mutex.RLock()
				count := len(spawnPoints)
				if count > 0 {
					_ = spawnPoints[0].NextSpawnAt
				}
				mutex.RUnlock()

				// Write operations (exclusive writers)
				if i%5 == 0 {
					mutex.Lock()
					if len(spawnPoints) > 0 {
						spawnPoints[0].NextSpawnAt = time.Now().Add(5 * time.Second)
					}
					mutex.Unlock()
				}

				// Small delay to increase chance of contention
				time.Sleep(time.Microsecond)
			}
		}(mapKey, mapIndex)
	}

	// Wait for all operations
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(10 * time.Second):
		t.Error("Test timed out - possible deadlock in multi-map concurrent access")
	}

	// Verify all maps are still intact
	for _, mapKey := range mapKeys {
		spawnPoints, exists := registry.GetSpawnPointsForMap(mapKey)
		if !exists || len(spawnPoints) != 2 {
			t.Errorf("Map %s should have 2 spawn points after concurrent access, got %d", mapKeyToString(mapKey), len(spawnPoints))
		}
	}
}

func TestCooldownEnforcementPreventsImmediateRespawn(t *testing.T) {
	// Test that cooldown enforcement prevents immediate re-spawning
	registry := GetRegistry()
	registry.Reset()

	// Create test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	now := time.Now()

	// Initialize registry with spawn points
	registry.registryMu.Lock()
	registry.spawnPointRegistry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 2, Template: 100101}, NextSpawnAt: now},
		{SpawnPoint: monster2.SpawnPoint{Id: 3, Template: 100102}, NextSpawnAt: now},
	}
	registry.spawnPointMu[mapKey] = &sync.RWMutex{}
	registry.registryMu.Unlock()

	spawnPoints, exists := registry.GetSpawnPointsForMap(mapKey)
	if !exists {
		t.Fatal("Registry should exist for map key")
	}

	registry.registryMu.RLock()
	mutex := registry.spawnPointMu[mapKey]
	registry.registryMu.RUnlock()

	// First spawn attempt - should succeed
	mutex.RLock()
	var eligibleIndices []int
	for i, sp := range spawnPoints {
		if sp.NextSpawnAt.Before(now) || sp.NextSpawnAt.Equal(now) {
			eligibleIndices = append(eligibleIndices, i)
		}
	}
	mutex.RUnlock()

	// Should have 3 eligible spawn points initially
	if len(eligibleIndices) != 3 {
		t.Errorf("Expected 3 eligible spawn points initially, got %d", len(eligibleIndices))
	}

	// Simulate spawning from first spawn point
	spawnIdx := eligibleIndices[0]
	mutex.Lock()
	spawnPoints[spawnIdx].NextSpawnAt = now.Add(5 * time.Second)
	mutex.Unlock()

	// Immediate re-spawn attempt - should be blocked
	mutex.RLock()
	var eligibleAfterSpawn []int
	for i, sp := range spawnPoints {
		if sp.NextSpawnAt.Before(now) || sp.NextSpawnAt.Equal(now) {
			eligibleAfterSpawn = append(eligibleAfterSpawn, i)
		}
	}
	mutex.RUnlock()

	// Should have 2 eligible spawn points after one is used
	if len(eligibleAfterSpawn) != 2 {
		t.Errorf("Expected 2 eligible spawn points after first spawn, got %d", len(eligibleAfterSpawn))
	}

	// Verify the spawned point is not in eligible list
	for _, idx := range eligibleAfterSpawn {
		if idx == spawnIdx {
			t.Errorf("Spawn point %d should not be eligible immediately after spawning", spawnIdx)
		}
	}

	// Simulate spawning from remaining spawn points
	for _, idx := range eligibleAfterSpawn {
		mutex.Lock()
		spawnPoints[idx].NextSpawnAt = now.Add(5 * time.Second)
		mutex.Unlock()
	}

	// All spawn points should now be on cooldown
	mutex.RLock()
	var eligibleAfterAllSpawns []int
	for i, sp := range spawnPoints {
		if sp.NextSpawnAt.Before(now) || sp.NextSpawnAt.Equal(now) {
			eligibleAfterAllSpawns = append(eligibleAfterAllSpawns, i)
		}
	}
	mutex.RUnlock()

	// Should have 0 eligible spawn points after all are used
	if len(eligibleAfterAllSpawns) != 0 {
		t.Errorf("Expected 0 eligible spawn points after all spawns, got %d", len(eligibleAfterAllSpawns))
	}

	// Verify all spawn points are on cooldown
	for i, sp := range spawnPoints {
		if sp.NextSpawnAt.Before(now) || sp.NextSpawnAt.Equal(now) {
			t.Errorf("Spawn point %d should be on cooldown", i)
		}
	}

	// Test that spawn points become eligible again after cooldown
	future := now.Add(6 * time.Second)
	var eligibleAfterCooldown []int
	for i, sp := range spawnPoints {
		if sp.NextSpawnAt.Before(future) {
			eligibleAfterCooldown = append(eligibleAfterCooldown, i)
		}
	}

	// All spawn points should be eligible again after cooldown
	if len(eligibleAfterCooldown) != 3 {
		t.Errorf("Expected 3 eligible spawn points after cooldown expires, got %d", len(eligibleAfterCooldown))
	}
}

func TestCooldownTimingAccuracy(t *testing.T) {
	// Test that cooldown timing is accurate to prevent early re-spawning
	registry := GetRegistry()
	registry.Reset()

	// Create test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	now := time.Now()

	// Initialize registry with one spawn point
	registry.registryMu.Lock()
	registry.spawnPointRegistry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: now},
	}
	registry.spawnPointMu[mapKey] = &sync.RWMutex{}
	registry.registryMu.Unlock()

	spawnPoints, exists := registry.GetSpawnPointsForMap(mapKey)
	if !exists {
		t.Fatal("Registry should exist for map key")
	}
	spawnPoint := spawnPoints[0]

	registry.registryMu.RLock()
	mutex := registry.spawnPointMu[mapKey]
	registry.registryMu.RUnlock()

	// Set cooldown to 5 seconds from now
	cooldownTime := now.Add(5 * time.Second)
	mutex.Lock()
	spawnPoint.NextSpawnAt = cooldownTime
	mutex.Unlock()

	// Test at various times before cooldown expires
	testTimes := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
		4 * time.Second,
		4*time.Second + 999*time.Millisecond, // Just before 5 seconds
	}

	for _, duration := range testTimes {
		testTime := now.Add(duration)
		if spawnPoint.NextSpawnAt.Before(testTime) || spawnPoint.NextSpawnAt.Equal(testTime) {
			t.Errorf("Spawn point should not be eligible at %v (cooldown expires at %v)", testTime, cooldownTime)
		}
	}

	// Test exactly at cooldown expiry
	if !spawnPoint.NextSpawnAt.Before(cooldownTime.Add(time.Millisecond)) {
		t.Error("Spawn point should be eligible after cooldown expires")
	}

	// Test after cooldown expires
	afterCooldown := now.Add(6 * time.Second)
	if !spawnPoint.NextSpawnAt.Before(afterCooldown) {
		t.Error("Spawn point should be eligible well after cooldown expires")
	}
}

func TestSequentialSpawnAttempts(t *testing.T) {
	// Test sequential spawn attempts to verify cooldown enforcement
	registry := GetRegistry()
	registry.Reset()

	// Create test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	baseTime := time.Now()

	// Initialize registry with spawn points
	registry.registryMu.Lock()
	registry.spawnPointRegistry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: baseTime},
		{SpawnPoint: monster2.SpawnPoint{Id: 2, Template: 100101}, NextSpawnAt: baseTime},
	}
	registry.spawnPointMu[mapKey] = &sync.RWMutex{}
	registry.registryMu.Unlock()

	spawnPoints, exists := registry.GetSpawnPointsForMap(mapKey)
	if !exists {
		t.Fatal("Registry should exist for map key")
	}

	registry.registryMu.RLock()
	mutex := registry.spawnPointMu[mapKey]
	registry.registryMu.RUnlock()

	// Track spawn attempts over time
	spawnAttempts := []struct {
		time     time.Time
		expected int // Expected number of eligible spawn points
	}{
		{baseTime, 2},                      // Both eligible initially
		{baseTime.Add(1 * time.Second), 2}, // Still both eligible
		{baseTime.Add(2 * time.Second), 2}, // Still both eligible
		{baseTime.Add(3 * time.Second), 2}, // Still both eligible
		{baseTime.Add(4 * time.Second), 2}, // Still both eligible
		{baseTime.Add(5 * time.Second), 2}, // Still both eligible
		{baseTime.Add(6 * time.Second), 2}, // Still both eligible
	}

	for i, attempt := range spawnAttempts {
		// Check eligibility at this time
		var eligible []int
		for idx, sp := range spawnPoints {
			if sp.NextSpawnAt.Before(attempt.time) || sp.NextSpawnAt.Equal(attempt.time) {
				eligible = append(eligible, idx)
			}
		}

		if len(eligible) != attempt.expected {
			t.Errorf("Attempt %d at time %v: expected %d eligible spawn points, got %d",
				i, attempt.time, attempt.expected, len(eligible))
		}

		// Simulate spawning if eligible points exist
		if len(eligible) > 0 {
			spawnIdx := eligible[0]
			mutex.Lock()
			spawnPoints[spawnIdx].NextSpawnAt = attempt.time.Add(5 * time.Second)
			mutex.Unlock()

			// Update expectations for subsequent attempts
			for j := i + 1; j < len(spawnAttempts); j++ {
				if spawnAttempts[j].time.Before(attempt.time.Add(5 * time.Second)) {
					spawnAttempts[j].expected--
				}
			}
		}
	}
}

func TestRapidSpawnAttempts(t *testing.T) {
	// Test rapid spawn attempts to ensure cooldown is enforced
	registry := GetRegistry()
	registry.Reset()

	// Create test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(1, 1, 100000000).Build()
	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	now := time.Now()

	// Initialize registry with single spawn point
	registry.registryMu.Lock()
	registry.spawnPointRegistry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: now},
	}
	registry.spawnPointMu[mapKey] = &sync.RWMutex{}
	registry.registryMu.Unlock()

	spawnPoints, exists := registry.GetSpawnPointsForMap(mapKey)
	if !exists {
		t.Fatal("Registry should exist for map key")
	}
	spawnPoint := spawnPoints[0]

	registry.registryMu.RLock()
	mutex := registry.spawnPointMu[mapKey]
	registry.registryMu.RUnlock()

	// First spawn should succeed
	if !spawnPoint.NextSpawnAt.Before(now) && !spawnPoint.NextSpawnAt.Equal(now) {
		t.Error("First spawn should be eligible")
	}

	// Simulate first spawn
	mutex.Lock()
	spawnPoint.NextSpawnAt = now.Add(5 * time.Second)
	mutex.Unlock()

	// Rapid subsequent spawn attempts should fail
	for i := 0; i < 10; i++ {
		attemptTime := now.Add(time.Duration(i) * 100 * time.Millisecond)
		if spawnPoint.NextSpawnAt.Before(attemptTime) || spawnPoint.NextSpawnAt.Equal(attemptTime) {
			t.Errorf("Rapid spawn attempt %d should be blocked by cooldown", i)
		}
	}

	// Spawn should be possible again after cooldown
	afterCooldown := now.Add(6 * time.Second)
	if !spawnPoint.NextSpawnAt.Before(afterCooldown) {
		t.Error("Spawn should be possible after cooldown expires")
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
	// Check all stored map keys to find a match by world/channel/map/instance (ignoring tenant)
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
	// Check all stored map keys to find a match by world/channel/map (ignoring tenant)
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

// TestSpawnMonsters_CooldownValidation tests the SpawnMonsters function to validate
// that the cooldown mechanism works correctly. It verifies that when x monsters
// are spawned, exactly x spawn points have their NextSpawnAt values updated.
func TestSpawnMonsters_CooldownValidation(t *testing.T) {
	// Create a test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	// Create mock processors
	mockCharProc := &mockCharacterProcessor{
		charactersInMap: make(map[character.MapKey][]uint32),
	}
	mockMonsterProc := &mockMonsterProcessor{
		monstersInMap: make(map[character.MapKey]int),
	}

	// Create 5 spawn points for the mock
	mockSpawnPoints := []monster2.SpawnPoint{
		{
			Id:       1,
			Template: 100100,
			MobTime:  10, // Spawnable
			X:        100,
			Y:        200,
			Fh:       10,
			Team:     0,
		},
		{
			Id:       2,
			Template: 100101,
			MobTime:  10, // Spawnable
			X:        150,
			Y:        230,
			Fh:       11,
			Team:     0,
		},
		{
			Id:       3,
			Template: 100102,
			MobTime:  10, // Spawnable
			X:        200,
			Y:        260,
			Fh:       12,
			Team:     0,
		},
		{
			Id:       4,
			Template: 100103,
			MobTime:  10, // Spawnable
			X:        250,
			Y:        290,
			Fh:       13,
			Team:     0,
		},
		{
			Id:       5,
			Template: 100104,
			MobTime:  10, // Spawnable
			X:        300,
			Y:        320,
			Fh:       14,
			Team:     0,
		},
	}

	// Create mock data processor
	mockDataProc := &mockDataProcessor{
		mockSpawnPoints: mockSpawnPoints,
	}

	// Create processor with mocked dependencies
	processor := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   te,
		dp:  mockDataProc,
		cp:  mockCharProc,
		mp:  mockMonsterProc,
	}

	// Reset the singleton registry for testing
	registry := GetRegistry()
	registry.Reset()

	// Setup test scenario
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()

	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	// Setup mock data: 2 characters in the map, 0 monsters currently
	mockCharProc.charactersInMap[mapKey] = []uint32{1001, 1002}
	mockMonsterProc.monstersInMap[mapKey] = 0

	// Calculate expected spawns using the same logic as getMonsterMax
	characterCount := 2
	spawnPointCount := len(mockSpawnPoints)
	spawnRate := 0.70 + (0.05 * float64(characterCount))
	expectedSpawns := int(math.Ceil(spawnRate * float64(spawnPointCount)))

	// Execute SpawnMonsters
	transactionId := uuid.New()

	// Catch any panics
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SpawnMonsters panicked: %v", r)
		}
	}()

	spawnErr := processor.SpawnMonsters(transactionId, f)

	// Verify no error occurred
	if spawnErr != nil {
		t.Errorf("SpawnMonsters should not return error, got: %v", spawnErr)
	}

	// Allow goroutines to complete
	time.Sleep(500 * time.Millisecond)

	// Get the spawn points from the singleton registry after execution
	registrySpawnPoints, exists := registry.GetSpawnPointsForMap(mapKey)
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
		// Check if cooldown was updated (NextSpawnAt should be approximately now + 3-5 seconds)
		// Since spawning happened a bit ago, the cooldown should be in the future but not too far
		if csp.NextSpawnAt.After(now.Add(2*time.Second)) && csp.NextSpawnAt.Before(now.Add(7*time.Second)) {
			updatedCount++
		}
	}

	// Verify that exactly the expected number of spawn points had their cooldowns updated
	if updatedCount != expectedSpawns {
		t.Errorf("Expected %d spawn points to have updated cooldowns, got %d", expectedSpawns, updatedCount)
	}

	// Verify that monsters were created
	createdMonsters := mockMonsterProc.GetCreatedMonsters()
	if len(createdMonsters) != expectedSpawns {
		t.Errorf("Expected %d monsters to be created, got %d", expectedSpawns, len(createdMonsters))
	}

	// Verify created monsters have correct parameters
	for _, monster := range createdMonsters {
		if monster.Field.WorldId() != worldId {
			t.Errorf("Created monster should have WorldId %d, got %d", worldId, monster.Field.WorldId())
		}
		if monster.Field.ChannelId() != channelId {
			t.Errorf("Created monster should have ChannelId %d, got %d", channelId, monster.Field.ChannelId())
		}
		if monster.Field.MapId() != mapId {
			t.Errorf("Created monster should have MapId %d, got %d", mapId, monster.Field.MapId())
		}

		// Verify monster was created from one of our spawn points
		found := false
		for _, sp := range mockSpawnPoints {
			if monster.MonsterId == sp.Template &&
				monster.X == sp.X &&
				monster.Y == sp.Y &&
				monster.Fh == sp.Fh &&
				monster.Team == sp.Team {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Created monster %+v does not match any spawn point", monster)
		}
	}
}

// TestSpawnMonsters_NoCharacters tests that no spawning occurs when no characters are present
func TestSpawnMonsters_NoCharacters(t *testing.T) {
	// Test that no spawning occurs when no characters are present
	registry := GetRegistry()
	registry.Reset()

	// Create a test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	// Create mock processors
	mockCharProc := &mockCharacterProcessor{
		charactersInMap: make(map[character.MapKey][]uint32),
	}
	mockMonsterProc := &mockMonsterProcessor{
		monstersInMap: make(map[character.MapKey]int),
	}

	// Create spawn point for the mock
	mockSpawnPoint := monster2.SpawnPoint{
		Id:       1,
		Template: 100100,
		MobTime:  10, // Spawnable
		X:        100,
		Y:        200,
		Fh:       10,
		Team:     0,
	}

	// Create mock data processor
	mockDataProc := &mockDataProcessor{
		mockSpawnPoints: []monster2.SpawnPoint{mockSpawnPoint},
	}

	// Create processor with mocked dependencies
	processor := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   te,
		dp:  mockDataProc,
		cp:  mockCharProc,
		mp:  mockMonsterProc,
	}

	// Setup test scenario
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()

	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	// Setup mock data: NO characters in the map
	mockCharProc.charactersInMap[mapKey] = []uint32{} // Empty
	mockMonsterProc.monstersInMap[mapKey] = 0
	mockMonsterProc.Reset()

	// Execute SpawnMonsters
	transactionId := uuid.New()
	err := processor.SpawnMonsters(transactionId, f)

	// Verify no error occurred
	if err != nil {
		t.Errorf("SpawnMonsters should not return error, got: %v", err)
	}

	// Allow goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Verify no monsters were created
	createdMonsters := mockMonsterProc.GetCreatedMonsters()
	if len(createdMonsters) != 0 {
		t.Errorf("Expected 0 monsters to be created when no characters present, got %d", len(createdMonsters))
	}

	// Verify that the registry was still initialized (even though no spawning occurred)
	registrySpawnPoints, exists := registry.GetSpawnPointsForMap(mapKey)
	if !exists {
		t.Error("Registry should be initialized even when no characters present")
	}

	if len(registrySpawnPoints) != 1 {
		t.Errorf("Expected 1 spawn point in registry, got %d", len(registrySpawnPoints))
	}
}

// TestSpawnMonsters_AllSpawnPointsOnCooldown tests behavior when all spawn points are on cooldown
func TestSpawnMonsters_AllSpawnPointsOnCooldown(t *testing.T) {
	// Test behavior when all spawn points are on cooldown
	registry := GetRegistry()
	registry.Reset()

	// Create a test tenant
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	// Create mock processors
	mockCharProc := &mockCharacterProcessor{
		charactersInMap: make(map[character.MapKey][]uint32),
	}
	mockMonsterProc := &mockMonsterProcessor{
		monstersInMap: make(map[character.MapKey]int),
	}

	// Create spawn points for the mock
	mockSpawnPoints := []monster2.SpawnPoint{
		{
			Id:       1,
			Template: 100100,
			MobTime:  10, // Spawnable
			X:        100,
			Y:        200,
			Fh:       10,
			Team:     0,
		},
		{
			Id:       2,
			Template: 100101,
			MobTime:  10, // Spawnable
			X:        150,
			Y:        200,
			Fh:       10,
			Team:     0,
		},
	}

	// Create mock data processor
	mockDataProc := &mockDataProcessor{
		mockSpawnPoints: mockSpawnPoints,
	}

	// Create processor with mocked dependencies
	processor := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   te,
		dp:  mockDataProc,
		cp:  mockCharProc,
		mp:  mockMonsterProc,
	}

	// Setup test scenario
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()

	mapKey := character.MapKey{
		Tenant: te,
		Field:  f,
	}

	// Setup mock data: characters in the map
	mockCharProc.charactersInMap[mapKey] = []uint32{1001, 1002}
	mockMonsterProc.monstersInMap[mapKey] = 0
	mockMonsterProc.Reset()

	// Pre-initialize the registry with all spawn points on cooldown
	now := time.Now()
	futureTime := now.Add(5 * time.Second)

	registry.registryMu.Lock()
	registry.spawnPointRegistry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: mockSpawnPoints[0], NextSpawnAt: futureTime}, // On cooldown
		{SpawnPoint: mockSpawnPoints[1], NextSpawnAt: futureTime}, // On cooldown
	}
	registry.spawnPointMu[mapKey] = &sync.RWMutex{}
	registry.registryMu.Unlock()

	// Record initial cooldown times
	initialSpawnPoints, _ := registry.GetSpawnPointsForMap(mapKey)
	initialCooldowns := []time.Time{
		initialSpawnPoints[0].NextSpawnAt,
		initialSpawnPoints[1].NextSpawnAt,
	}

	// Execute SpawnMonsters
	transactionId := uuid.New()
	err := processor.SpawnMonsters(transactionId, f)

	// Verify no error occurred
	if err != nil {
		t.Errorf("SpawnMonsters should not return error, got: %v", err)
	}

	// Allow goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Verify no cooldown updates (all spawn points were on cooldown)
	finalSpawnPoints, _ := registry.GetSpawnPointsForMap(mapKey)
	for i, sp := range finalSpawnPoints {
		if !sp.NextSpawnAt.Equal(initialCooldowns[i]) {
			t.Errorf("Spawn point %d cooldown should not be updated when on cooldown. Initial: %v, Current: %v",
				i, initialCooldowns[i], sp.NextSpawnAt)
		}
	}

	// Verify no monsters were created
	createdMonsters := mockMonsterProc.GetCreatedMonsters()
	if len(createdMonsters) != 0 {
		t.Errorf("Expected 0 monsters to be created when all spawn points on cooldown, got %d", len(createdMonsters))
	}
}
