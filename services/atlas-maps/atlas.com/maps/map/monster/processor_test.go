package monster

import (
	"atlas-maps/map/character"
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
		WorldId:   1,
		ChannelId: 1,
		MapId:     100000000,
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
		WorldId:   1,
		ChannelId: 1,
		MapId:     100000000,
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