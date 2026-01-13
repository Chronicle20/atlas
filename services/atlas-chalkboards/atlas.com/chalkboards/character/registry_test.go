package character

import (
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func newTestTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func newTestMapKey(t tenant.Model) MapKey {
	return MapKey{
		Tenant:    t,
		WorldId:   world.Id(0),
		ChannelId: channel.Id(1),
		MapId:     _map.Id(100000000),
	}
}

func resetRegistry() {
	r := getRegistry()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister = make(map[MapKey][]uint32)
	r.mapLocks = make(map[MapKey]*sync.RWMutex)
}

func TestRegistryGetInMapEmpty(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()
	key := newTestMapKey(ten)

	result := getRegistry().GetInMap(key)
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}
}

func TestRegistryAddCharacter(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()
	key := newTestMapKey(ten)
	characterId := uint32(12345)

	getRegistry().AddCharacter(key, characterId)

	result := getRegistry().GetInMap(key)
	if len(result) != 1 {
		t.Fatalf("Expected 1 character, got %d", len(result))
	}
	if result[0] != characterId {
		t.Errorf("Expected character %d, got %d", characterId, result[0])
	}
}

func TestRegistryAddCharacterDuplicate(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()
	key := newTestMapKey(ten)
	characterId := uint32(12345)

	getRegistry().AddCharacter(key, characterId)
	getRegistry().AddCharacter(key, characterId)

	result := getRegistry().GetInMap(key)
	if len(result) != 1 {
		t.Errorf("Expected 1 character (no duplicates), got %d", len(result))
	}
}

func TestRegistryAddMultipleCharacters(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()
	key := newTestMapKey(ten)

	getRegistry().AddCharacter(key, 1)
	getRegistry().AddCharacter(key, 2)
	getRegistry().AddCharacter(key, 3)

	result := getRegistry().GetInMap(key)
	if len(result) != 3 {
		t.Errorf("Expected 3 characters, got %d", len(result))
	}
}

func TestRegistryRemoveCharacter(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()
	key := newTestMapKey(ten)
	characterId := uint32(12345)

	getRegistry().AddCharacter(key, characterId)
	getRegistry().RemoveCharacter(key, characterId)

	result := getRegistry().GetInMap(key)
	if len(result) != 0 {
		t.Errorf("Expected empty slice after removal, got %v", result)
	}
}

func TestRegistryRemoveCharacterNotExists(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()
	key := newTestMapKey(ten)

	// Should not panic when removing non-existent character
	getRegistry().RemoveCharacter(key, 99999)
}

func TestRegistryRemoveCharacterPreservesOthers(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()
	key := newTestMapKey(ten)

	getRegistry().AddCharacter(key, 1)
	getRegistry().AddCharacter(key, 2)
	getRegistry().AddCharacter(key, 3)
	getRegistry().RemoveCharacter(key, 2)

	result := getRegistry().GetInMap(key)
	if len(result) != 2 {
		t.Fatalf("Expected 2 characters, got %d", len(result))
	}

	found1, found3 := false, false
	for _, id := range result {
		if id == 1 {
			found1 = true
		}
		if id == 3 {
			found3 = true
		}
		if id == 2 {
			t.Error("Character 2 should have been removed")
		}
	}
	if !found1 || !found3 {
		t.Error("Expected characters 1 and 3 to remain")
	}
}

func TestRegistryTenantIsolation(t *testing.T) {
	resetRegistry()
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	key1 := MapKey{Tenant: tenant1, WorldId: 0, ChannelId: 1, MapId: 100000000}
	key2 := MapKey{Tenant: tenant2, WorldId: 0, ChannelId: 1, MapId: 100000000}

	getRegistry().AddCharacter(key1, 1)
	getRegistry().AddCharacter(key2, 2)

	result1 := getRegistry().GetInMap(key1)
	result2 := getRegistry().GetInMap(key2)

	if len(result1) != 1 || result1[0] != 1 {
		t.Errorf("Tenant1 expected [1], got %v", result1)
	}
	if len(result2) != 1 || result2[0] != 2 {
		t.Errorf("Tenant2 expected [2], got %v", result2)
	}

	// Removing from tenant1 should not affect tenant2
	getRegistry().RemoveCharacter(key1, 1)
	result2After := getRegistry().GetInMap(key2)
	if len(result2After) != 1 {
		t.Error("Tenant2 data should not be affected by tenant1 removal")
	}
}

func TestRegistryMapIsolation(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()

	key1 := MapKey{Tenant: ten, WorldId: 0, ChannelId: 1, MapId: 100000000}
	key2 := MapKey{Tenant: ten, WorldId: 0, ChannelId: 1, MapId: 200000000}

	getRegistry().AddCharacter(key1, 1)
	getRegistry().AddCharacter(key2, 2)

	result1 := getRegistry().GetInMap(key1)
	result2 := getRegistry().GetInMap(key2)

	if len(result1) != 1 || result1[0] != 1 {
		t.Errorf("Map1 expected [1], got %v", result1)
	}
	if len(result2) != 1 || result2[0] != 2 {
		t.Errorf("Map2 expected [2], got %v", result2)
	}
}

func TestRegistryConcurrent(t *testing.T) {
	resetRegistry()
	ten := newTestTenant()
	key := newTestMapKey(ten)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			getRegistry().AddCharacter(key, characterId)
			getRegistry().GetInMap(key)
			getRegistry().RemoveCharacter(key, characterId)
		}(i)
	}

	wg.Wait()
}
