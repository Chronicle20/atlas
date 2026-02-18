package character

import (
	"context"
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func setupCharacterTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func sampleTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func sampleMapKey(t tenant.Model, worldId world.Id, channelId channel.Id, mapId _map.Id) MapKey {
	return MapKey{
		Tenant: t,
		Field:  field.NewBuilder(worldId, channelId, mapId).Build(),
	}
}

func TestRegistryGetInMapEmpty(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	result := getRegistry().GetInMap(ctx, key)
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}
}

func TestRegistryAddCharacter(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)
	characterId := uint32(12345)

	getRegistry().AddCharacter(ctx, key, characterId)

	result := getRegistry().GetInMap(ctx, key)
	if len(result) != 1 {
		t.Fatalf("Expected 1 character, got %d", len(result))
	}
	if result[0] != characterId {
		t.Errorf("Expected character %d, got %d", characterId, result[0])
	}
}

func TestRegistryAddCharacterDuplicate(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)
	characterId := uint32(12345)

	getRegistry().AddCharacter(ctx, key, characterId)
	getRegistry().AddCharacter(ctx, key, characterId)

	result := getRegistry().GetInMap(ctx, key)
	if len(result) != 1 {
		t.Errorf("Expected 1 character (no duplicates), got %d", len(result))
	}
}

func TestRegistryAddMultipleCharacters(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	getRegistry().AddCharacter(ctx, key, 1)
	getRegistry().AddCharacter(ctx, key, 2)
	getRegistry().AddCharacter(ctx, key, 3)

	result := getRegistry().GetInMap(ctx, key)
	if len(result) != 3 {
		t.Errorf("Expected 3 characters, got %d", len(result))
	}
}

func TestRegistryRemoveCharacter(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)
	characterId := uint32(12345)

	getRegistry().AddCharacter(ctx, key, characterId)
	getRegistry().RemoveCharacter(ctx, key, characterId)

	result := getRegistry().GetInMap(ctx, key)
	if len(result) != 0 {
		t.Errorf("Expected empty slice after removal, got %v", result)
	}
}

func TestRegistryRemoveCharacterNotExists(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	getRegistry().RemoveCharacter(ctx, key, 99999)
}

func TestRegistryRemoveCharacterPreservesOthers(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	getRegistry().AddCharacter(ctx, key, 1)
	getRegistry().AddCharacter(ctx, key, 2)
	getRegistry().AddCharacter(ctx, key, 3)
	getRegistry().RemoveCharacter(ctx, key, 2)

	result := getRegistry().GetInMap(ctx, key)
	if len(result) != 2 {
		t.Fatalf("Expected 2 characters, got %d", len(result))
	}

	charMap := make(map[uint32]bool)
	for _, c := range result {
		charMap[c] = true
	}
	if !charMap[1] || !charMap[3] {
		t.Error("Expected characters 1 and 3 to remain")
	}
	if charMap[2] {
		t.Error("Character 2 should have been removed")
	}
}

func TestRegistryTenantIsolation(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	key1 := sampleMapKey(tenant1, 0, 1, 100000000)
	key2 := sampleMapKey(tenant2, 0, 1, 100000000)

	getRegistry().AddCharacter(ctx, key1, 1)
	getRegistry().AddCharacter(ctx, key2, 2)

	result1 := getRegistry().GetInMap(ctx, key1)
	result2 := getRegistry().GetInMap(ctx, key2)

	if len(result1) != 1 || result1[0] != 1 {
		t.Errorf("Tenant1 expected [1], got %v", result1)
	}
	if len(result2) != 1 || result2[0] != 2 {
		t.Errorf("Tenant2 expected [2], got %v", result2)
	}

	getRegistry().RemoveCharacter(ctx, key1, 1)
	result2After := getRegistry().GetInMap(ctx, key2)
	if len(result2After) != 1 {
		t.Error("Tenant2 data should not be affected by tenant1 removal")
	}
}

func TestRegistryMapIsolation(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()

	key1 := sampleMapKey(st, 0, 1, 100000000)
	key2 := sampleMapKey(st, 0, 1, 200000000)

	getRegistry().AddCharacter(ctx, key1, 1)
	getRegistry().AddCharacter(ctx, key2, 2)

	result1 := getRegistry().GetInMap(ctx, key1)
	result2 := getRegistry().GetInMap(ctx, key2)

	if len(result1) != 1 || result1[0] != 1 {
		t.Errorf("Map1 expected [1], got %v", result1)
	}
	if len(result2) != 1 || result2[0] != 2 {
		t.Errorf("Map2 expected [2], got %v", result2)
	}
}

func TestRegistryConcurrent(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()
	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			getRegistry().AddCharacter(ctx, key, characterId)
			getRegistry().GetInMap(ctx, key)
			getRegistry().RemoveCharacter(ctx, key, characterId)
		}(i)
	}

	wg.Wait()
}
