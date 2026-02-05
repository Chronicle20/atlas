package character

import (
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testMapKey(worldId world.Id, channelId channel.Id, mapId _map.Id) MapKey {
	return MapKey{
		Tenant: testTenant(),
		Field:  field.NewBuilder(worldId, channelId, mapId).Build(),
	}
}

func TestRegistry_AddCharacter_And_GetMap(t *testing.T) {
	r := getRegistry()
	characterId := uint32(12345)
	mk := testMapKey(1, 2, 100000000)

	// Clean up before test
	r.RemoveCharacter(characterId)

	r.AddCharacter(mk, characterId)

	result, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("expected to find character in registry")
	}

	if result.Field.WorldId() != mk.Field.WorldId() {
		t.Errorf("expected WorldId %d, got %d", mk.Field.WorldId(), result.Field.WorldId())
	}
	if result.Field.ChannelId() != mk.Field.ChannelId() {
		t.Errorf("expected ChannelId %d, got %d", mk.Field.ChannelId(), result.Field.ChannelId())
	}
	if result.Field.MapId() != mk.Field.MapId() {
		t.Errorf("expected MapId %d, got %d", mk.Field.MapId(), result.Field.MapId())
	}

	// Clean up after test
	r.RemoveCharacter(characterId)
}

func TestRegistry_RemoveCharacter(t *testing.T) {
	r := getRegistry()
	characterId := uint32(12346)
	mk := testMapKey(1, 2, 100000000)

	// Add character first
	r.AddCharacter(mk, characterId)

	// Verify it exists
	_, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should exist before removal")
	}

	// Remove character
	r.RemoveCharacter(characterId)

	// Verify it's gone
	_, ok = r.GetMap(characterId)
	if ok {
		t.Error("character should not exist after removal")
	}
}

func TestRegistry_GetMap_NotFound(t *testing.T) {
	r := getRegistry()
	nonExistentId := uint32(99999999)

	// Ensure it doesn't exist
	r.RemoveCharacter(nonExistentId)

	_, ok := r.GetMap(nonExistentId)
	if ok {
		t.Error("expected not to find non-existent character")
	}
}

func TestRegistry_AddCharacter_OverwritesExisting(t *testing.T) {
	r := getRegistry()
	characterId := uint32(12347)
	mk1 := testMapKey(1, 1, 100000000)
	mk2 := testMapKey(2, 3, 200000000)

	// Clean up
	r.RemoveCharacter(characterId)

	// Add with first map key
	r.AddCharacter(mk1, characterId)

	// Add with second map key (should overwrite)
	r.AddCharacter(mk2, characterId)

	result, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("expected to find character")
	}

	if result.Field.WorldId() != mk2.Field.WorldId() {
		t.Errorf("expected WorldId %d, got %d", mk2.Field.WorldId(), result.Field.WorldId())
	}
	if result.Field.ChannelId() != mk2.Field.ChannelId() {
		t.Errorf("expected ChannelId %d, got %d", mk2.Field.ChannelId(), result.Field.ChannelId())
	}
	if result.Field.MapId() != mk2.Field.MapId() {
		t.Errorf("expected MapId %d, got %d", mk2.Field.MapId(), result.Field.MapId())
	}

	// Clean up
	r.RemoveCharacter(characterId)
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := getRegistry()
	baseCharacterId := uint32(50000)
	numGoroutines := 100
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineId int) {
			defer wg.Done()
			characterId := baseCharacterId + uint32(goroutineId)
			mk := testMapKey(world.Id(goroutineId%256), channel.Id(goroutineId%20), _map.Id(100000000+goroutineId))

			for j := 0; j < numOperations; j++ {
				// Add
				r.AddCharacter(mk, characterId)

				// Get
				_, _ = r.GetMap(characterId)

				// Remove
				r.RemoveCharacter(characterId)
			}
		}(i)
	}

	wg.Wait()

	// Clean up all test characters
	for i := 0; i < numGoroutines; i++ {
		r.RemoveCharacter(baseCharacterId + uint32(i))
	}
}

func TestRegistry_Singleton(t *testing.T) {
	r1 := getRegistry()
	r2 := getRegistry()

	if r1 != r2 {
		t.Error("getRegistry should return the same instance")
	}
}
