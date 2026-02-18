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

func TestRegistry_AddCharacter(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)
	characterId := uint32(12345)

	getRegistry().AddCharacter(ctx, key, characterId)

	chars := getRegistry().GetInMap(ctx, key)
	if len(chars) != 1 {
		t.Fatalf("Expected 1 character in map, got %d", len(chars))
	}

	if chars[0] != characterId {
		t.Errorf("Expected character %d, got %d", characterId, chars[0])
	}
}

func TestRegistry_AddCharacter_Duplicate(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)
	characterId := uint32(12345)

	// Add same character twice — Redis SADD handles dedup
	getRegistry().AddCharacter(ctx, key, characterId)
	getRegistry().AddCharacter(ctx, key, characterId)

	chars := getRegistry().GetInMap(ctx, key)
	if len(chars) != 1 {
		t.Fatalf("Expected 1 character (no duplicates), got %d", len(chars))
	}
}

func TestRegistry_AddCharacter_Multiple(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	characterIds := []uint32{100, 200, 300}
	for _, cid := range characterIds {
		getRegistry().AddCharacter(ctx, key, cid)
	}

	chars := getRegistry().GetInMap(ctx, key)
	if len(chars) != len(characterIds) {
		t.Fatalf("Expected %d characters, got %d", len(characterIds), len(chars))
	}

	// Verify all characters are present
	charMap := make(map[uint32]bool)
	for _, c := range chars {
		charMap[c] = true
	}
	for _, cid := range characterIds {
		if !charMap[cid] {
			t.Errorf("Expected character %d to be in map", cid)
		}
	}
}

func TestRegistry_RemoveCharacter(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)
	characterId := uint32(12345)

	getRegistry().AddCharacter(ctx, key, characterId)

	// Verify added
	chars := getRegistry().GetInMap(ctx, key)
	if len(chars) != 1 {
		t.Fatalf("Expected 1 character after add, got %d", len(chars))
	}

	// Remove character
	getRegistry().RemoveCharacter(ctx, key, characterId)

	chars = getRegistry().GetInMap(ctx, key)
	if len(chars) != 0 {
		t.Fatalf("Expected 0 characters after remove, got %d", len(chars))
	}
}

func TestRegistry_RemoveCharacter_NotExists(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)
	characterId := uint32(99999)

	// Remove non-existent character should not panic
	getRegistry().RemoveCharacter(ctx, key, characterId)

	chars := getRegistry().GetInMap(ctx, key)
	if len(chars) != 0 {
		t.Fatalf("Expected 0 characters, got %d", len(chars))
	}
}

func TestRegistry_RemoveCharacter_PreservesOthers(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	getRegistry().AddCharacter(ctx, key, 100)
	getRegistry().AddCharacter(ctx, key, 200)
	getRegistry().AddCharacter(ctx, key, 300)

	// Remove middle character
	getRegistry().RemoveCharacter(ctx, key, 200)

	chars := getRegistry().GetInMap(ctx, key)
	if len(chars) != 2 {
		t.Fatalf("Expected 2 characters after remove, got %d", len(chars))
	}

	charMap := make(map[uint32]bool)
	for _, c := range chars {
		charMap[c] = true
	}

	if !charMap[100] {
		t.Error("Expected character 100 to remain")
	}
	if charMap[200] {
		t.Error("Expected character 200 to be removed")
	}
	if !charMap[300] {
		t.Error("Expected character 300 to remain")
	}
}

func TestRegistry_GetInMap_Empty(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	chars := getRegistry().GetInMap(ctx, key)
	if chars != nil && len(chars) != 0 {
		t.Fatalf("Expected empty or nil slice for non-existent map, got %v", chars)
	}
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	key1 := sampleMapKey(tenant1, 0, 1, 100000000)
	key2 := sampleMapKey(tenant2, 0, 1, 100000000)

	getRegistry().AddCharacter(ctx, key1, 100)
	getRegistry().AddCharacter(ctx, key2, 200)

	chars1 := getRegistry().GetInMap(ctx, key1)
	chars2 := getRegistry().GetInMap(ctx, key2)

	if len(chars1) != 1 || chars1[0] != 100 {
		t.Errorf("Tenant1 should have character 100, got %v", chars1)
	}

	if len(chars2) != 1 || chars2[0] != 200 {
		t.Errorf("Tenant2 should have character 200, got %v", chars2)
	}
}

func TestRegistry_MapIsolation(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key1 := sampleMapKey(st, 0, 1, 100000000)
	key2 := sampleMapKey(st, 0, 1, 200000000)

	getRegistry().AddCharacter(ctx, key1, 100)
	getRegistry().AddCharacter(ctx, key2, 200)

	chars1 := getRegistry().GetInMap(ctx, key1)
	chars2 := getRegistry().GetInMap(ctx, key2)

	if len(chars1) != 1 || chars1[0] != 100 {
		t.Errorf("Map1 should have character 100, got %v", chars1)
	}

	if len(chars2) != 1 || chars2[0] != 200 {
		t.Errorf("Map2 should have character 200, got %v", chars2)
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	key := sampleMapKey(st, 0, 1, 100000000)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent adds
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			getRegistry().AddCharacter(ctx, key, uint32(id))
		}(i)
	}

	wg.Wait()

	chars := getRegistry().GetInMap(ctx, key)
	if len(chars) != iterations {
		t.Errorf("Expected %d characters, got %d", iterations, len(chars))
	}

	// Concurrent reads and removes
	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			getRegistry().GetInMap(ctx, key)
		}()
		go func(id int) {
			defer wg.Done()
			getRegistry().RemoveCharacter(ctx, key, uint32(id))
		}(i)
	}

	wg.Wait()

	chars = getRegistry().GetInMap(ctx, key)
	if len(chars) != 0 {
		t.Errorf("Expected 0 characters after all removes, got %d", len(chars))
	}
}

func TestRegistry_ConcurrentDifferentMaps(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := context.Background()

	st := sampleTenant()
	numMaps := 10
	charsPerMap := 50

	var wg sync.WaitGroup

	// Concurrent adds to different maps
	for m := 0; m < numMaps; m++ {
		key := sampleMapKey(st, 0, 1, _map.Id(100000000+m))
		for c := 0; c < charsPerMap; c++ {
			wg.Add(1)
			go func(k MapKey, charId uint32) {
				defer wg.Done()
				getRegistry().AddCharacter(ctx, k, charId)
			}(key, uint32(m*1000+c))
		}
	}

	wg.Wait()

	// Verify each map has correct number of characters
	for m := 0; m < numMaps; m++ {
		key := sampleMapKey(st, 0, 1, _map.Id(100000000+m))
		chars := getRegistry().GetInMap(ctx, key)
		if len(chars) != charsPerMap {
			t.Errorf("Map %d: expected %d characters, got %d", m, charsPerMap, len(chars))
		}
	}
}
