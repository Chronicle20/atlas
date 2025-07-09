package monster

import (
	"atlas-maps/map/character"
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"sync"
	"testing"
	"time"
)

func TestCooldownSpawnPoint_Creation(t *testing.T) {
	sp := SpawnPoint{
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
		{SpawnPoint: SpawnPoint{Id: 1}, NextSpawnAt: now.Add(-1 * time.Second)}, // Eligible
		{SpawnPoint: SpawnPoint{Id: 2}, NextSpawnAt: now.Add(3 * time.Second)},  // On cooldown
		{SpawnPoint: SpawnPoint{Id: 3}, NextSpawnAt: now},                       // Eligible (equal time)
		{SpawnPoint: SpawnPoint{Id: 4}, NextSpawnAt: now.Add(-5 * time.Second)}, // Eligible
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
		SpawnPoint:  SpawnPoint{Id: 1},
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
	
	originalSpawnPoints := []SpawnPoint{
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
	
	mapKey := character.MapKey{
		WorldId:   world.Id(1),
		ChannelId: channel.Id(1),
		MapId:     _map.Id(100000000),
	}
	
	// Initialize spawn points
	registry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: SpawnPoint{Id: 1}, NextSpawnAt: now.Add(-1 * time.Second)}, // Eligible
		{SpawnPoint: SpawnPoint{Id: 2}, NextSpawnAt: now.Add(3 * time.Second)},  // On cooldown
		{SpawnPoint: SpawnPoint{Id: 3}, NextSpawnAt: now},                       // Eligible
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
	
	mapKey := character.MapKey{
		WorldId:   world.Id(1),
		ChannelId: channel.Id(1),
		MapId:     _map.Id(100000000),
	}
	
	// Initialize spawn points
	now := time.Now()
	registry[mapKey] = []*CooldownSpawnPoint{
		{SpawnPoint: SpawnPoint{Id: 1}, NextSpawnAt: now},
		{SpawnPoint: SpawnPoint{Id: 2}, NextSpawnAt: now},
		{SpawnPoint: SpawnPoint{Id: 3}, NextSpawnAt: now},
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
	processor := &ProcessorImpl{
		spawnPointRegistry: make(map[character.MapKey][]*CooldownSpawnPoint),
		spawnPointMu:       make(map[character.MapKey]*sync.RWMutex),
	}
	
	// Create multiple map keys for different maps
	mapKeys := []character.MapKey{
		{WorldId: world.Id(1), ChannelId: channel.Id(1), MapId: _map.Id(100000000)},
		{WorldId: world.Id(1), ChannelId: channel.Id(1), MapId: _map.Id(100000001)},
		{WorldId: world.Id(1), ChannelId: channel.Id(2), MapId: _map.Id(100000000)},
		{WorldId: world.Id(2), ChannelId: channel.Id(1), MapId: _map.Id(100000000)},
		{WorldId: world.Id(1), ChannelId: channel.Id(1), MapId: _map.Id(100000002)},
	}
	
	now := time.Now()
	
	// Initialize registry for each map
	for i, mapKey := range mapKeys {
		processor.spawnPointRegistry[mapKey] = []*CooldownSpawnPoint{
			{SpawnPoint: SpawnPoint{Id: uint32(i*10 + 1)}, NextSpawnAt: now},
			{SpawnPoint: SpawnPoint{Id: uint32(i*10 + 2)}, NextSpawnAt: now},
			{SpawnPoint: SpawnPoint{Id: uint32(i*10 + 3)}, NextSpawnAt: now},
		}
		processor.spawnPointMu[mapKey] = &sync.RWMutex{}
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
				
				mutex := processor.spawnPointMu[mk]
				spawnPoints := processor.spawnPointRegistry[mk]
				
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
		if _, exists := processor.spawnPointRegistry[mapKey]; !exists {
			t.Errorf("Registry for map %s should exist", mapKeyToString(mapKey))
		}
		if _, exists := processor.spawnPointMu[mapKey]; !exists {
			t.Errorf("Mutex for map %s should exist", mapKeyToString(mapKey))
		}
	}
	
	// Verify that different maps have different registry entries
	registryCount := len(processor.spawnPointRegistry)
	if registryCount != len(mapKeys) {
		t.Errorf("Expected %d registry entries, got %d", len(mapKeys), registryCount)
	}
	
	mutexCount := len(processor.spawnPointMu)
	if mutexCount != len(mapKeys) {
		t.Errorf("Expected %d mutex entries, got %d", len(mapKeys), mutexCount)
	}
}

func TestMapKeyIsolation(t *testing.T) {
	// Test that different MapKeys maintain separate registries
	processor := &ProcessorImpl{
		spawnPointRegistry: make(map[character.MapKey][]*CooldownSpawnPoint),
		spawnPointMu:       make(map[character.MapKey]*sync.RWMutex),
	}
	
	// Create different map keys
	mapKey1 := character.MapKey{WorldId: world.Id(1), ChannelId: channel.Id(1), MapId: _map.Id(100000000)}
	mapKey2 := character.MapKey{WorldId: world.Id(1), ChannelId: channel.Id(1), MapId: _map.Id(100000001)}
	mapKey3 := character.MapKey{WorldId: world.Id(1), ChannelId: channel.Id(2), MapId: _map.Id(100000000)}
	mapKey4 := character.MapKey{WorldId: world.Id(2), ChannelId: channel.Id(1), MapId: _map.Id(100000000)}
	
	now := time.Now()
	
	// Initialize registries with different spawn points
	processor.spawnPointRegistry[mapKey1] = []*CooldownSpawnPoint{
		{SpawnPoint: SpawnPoint{Id: 1}, NextSpawnAt: now},
	}
	processor.spawnPointMu[mapKey1] = &sync.RWMutex{}
	
	processor.spawnPointRegistry[mapKey2] = []*CooldownSpawnPoint{
		{SpawnPoint: SpawnPoint{Id: 2}, NextSpawnAt: now},
		{SpawnPoint: SpawnPoint{Id: 3}, NextSpawnAt: now},
	}
	processor.spawnPointMu[mapKey2] = &sync.RWMutex{}
	
	processor.spawnPointRegistry[mapKey3] = []*CooldownSpawnPoint{
		{SpawnPoint: SpawnPoint{Id: 4}, NextSpawnAt: now},
		{SpawnPoint: SpawnPoint{Id: 5}, NextSpawnAt: now},
		{SpawnPoint: SpawnPoint{Id: 6}, NextSpawnAt: now},
	}
	processor.spawnPointMu[mapKey3] = &sync.RWMutex{}
	
	processor.spawnPointRegistry[mapKey4] = []*CooldownSpawnPoint{
		{SpawnPoint: SpawnPoint{Id: 7}, NextSpawnAt: now},
		{SpawnPoint: SpawnPoint{Id: 8}, NextSpawnAt: now},
		{SpawnPoint: SpawnPoint{Id: 9}, NextSpawnAt: now},
		{SpawnPoint: SpawnPoint{Id: 10}, NextSpawnAt: now},
	}
	processor.spawnPointMu[mapKey4] = &sync.RWMutex{}
	
	// Verify that each map has the correct number of spawn points
	if len(processor.spawnPointRegistry[mapKey1]) != 1 {
		t.Errorf("MapKey1 should have 1 spawn point, got %d", len(processor.spawnPointRegistry[mapKey1]))
	}
	
	if len(processor.spawnPointRegistry[mapKey2]) != 2 {
		t.Errorf("MapKey2 should have 2 spawn points, got %d", len(processor.spawnPointRegistry[mapKey2]))
	}
	
	if len(processor.spawnPointRegistry[mapKey3]) != 3 {
		t.Errorf("MapKey3 should have 3 spawn points, got %d", len(processor.spawnPointRegistry[mapKey3]))
	}
	
	if len(processor.spawnPointRegistry[mapKey4]) != 4 {
		t.Errorf("MapKey4 should have 4 spawn points, got %d", len(processor.spawnPointRegistry[mapKey4]))
	}
	
	// Verify that spawn points are isolated (different IDs)
	if processor.spawnPointRegistry[mapKey1][0].SpawnPoint.Id != 1 {
		t.Errorf("MapKey1 spawn point should have ID 1, got %d", processor.spawnPointRegistry[mapKey1][0].SpawnPoint.Id)
	}
	
	if processor.spawnPointRegistry[mapKey2][0].SpawnPoint.Id != 2 {
		t.Errorf("MapKey2 first spawn point should have ID 2, got %d", processor.spawnPointRegistry[mapKey2][0].SpawnPoint.Id)
	}
	
	// Verify that modifying one map doesn't affect others
	processor.spawnPointMu[mapKey1].Lock()
	processor.spawnPointRegistry[mapKey1][0].NextSpawnAt = now.Add(10 * time.Second)
	processor.spawnPointMu[mapKey1].Unlock()
	
	// Check that other maps are unaffected
	if processor.spawnPointRegistry[mapKey2][0].NextSpawnAt.After(now.Add(time.Second)) {
		t.Error("MapKey2 spawn points should not be affected by MapKey1 modifications")
	}
	
	if processor.spawnPointRegistry[mapKey3][0].NextSpawnAt.After(now.Add(time.Second)) {
		t.Error("MapKey3 spawn points should not be affected by MapKey1 modifications")
	}
	
	if processor.spawnPointRegistry[mapKey4][0].NextSpawnAt.After(now.Add(time.Second)) {
		t.Error("MapKey4 spawn points should not be affected by MapKey1 modifications")
	}
}

func TestMultiMapConcurrentAccess(t *testing.T) {
	// Test that concurrent access to different maps doesn't interfere
	processor := &ProcessorImpl{
		spawnPointRegistry: make(map[character.MapKey][]*CooldownSpawnPoint),
		spawnPointMu:       make(map[character.MapKey]*sync.RWMutex),
	}
	
	// Create multiple maps
	numMaps := 5
	mapKeys := make([]character.MapKey, numMaps)
	for i := 0; i < numMaps; i++ {
		mapKeys[i] = character.MapKey{
			WorldId:   world.Id(1),
			ChannelId: channel.Id(1),
			MapId:     _map.Id(100000000 + i),
		}
	}
	
	now := time.Now()
	
	// Initialize all maps
	for i, mapKey := range mapKeys {
		processor.spawnPointRegistry[mapKey] = []*CooldownSpawnPoint{
			{SpawnPoint: SpawnPoint{Id: uint32(i*10 + 1)}, NextSpawnAt: now},
			{SpawnPoint: SpawnPoint{Id: uint32(i*10 + 2)}, NextSpawnAt: now},
		}
		processor.spawnPointMu[mapKey] = &sync.RWMutex{}
	}
	
	var wg sync.WaitGroup
	const iterations = 50
	
	// Start concurrent operations on each map
	for mapIndex, mapKey := range mapKeys {
		wg.Add(1)
		go func(mk character.MapKey, index int) {
			defer wg.Done()
			
			for i := 0; i < iterations; i++ {
				mutex := processor.spawnPointMu[mk]
				spawnPoints := processor.spawnPointRegistry[mk]
				
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
		if len(processor.spawnPointRegistry[mapKey]) != 2 {
			t.Errorf("Map %s should have 2 spawn points after concurrent access", mapKeyToString(mapKey))
		}
	}
}

// Helper function to convert MapKey to string for testing
func mapKeyToString(mk character.MapKey) string {
	return fmt.Sprintf("W%d-C%d-M%d", mk.WorldId, mk.ChannelId, mk.MapId)
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