package character

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

func createTestContext() (context.Context, tenant.Model) {
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(ctx, te), te
}

func TestProcessorImpl_Enter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, te := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := _map.Id(100000000)
	instance := uuid.Nil
	characterId := uint32(12345)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Enter character
	p.Enter(transactionId, f, characterId)

	// Verify character is in map
	key := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance}
	characters := getRegistry().GetInMap(key)

	if len(characters) != 1 {
		t.Fatalf("Expected 1 character, got %d", len(characters))
	}
	if characters[0] != characterId {
		t.Errorf("Expected character ID %d, got %d", characterId, characters[0])
	}
}

func TestProcessorImpl_Enter_Multiple(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, te := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(2)
	channelId := channel.Id(2)
	mapId := _map.Id(100000001)
	instance := uuid.Nil
	characterId1 := uint32(12345)
	characterId2 := uint32(67890)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Enter multiple characters
	p.Enter(transactionId, f, characterId1)
	p.Enter(transactionId, f, characterId2)

	// Verify both characters are in map
	key := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance}
	characters := getRegistry().GetInMap(key)

	if len(characters) != 2 {
		t.Fatalf("Expected 2 characters, got %d", len(characters))
	}

	found1, found2 := false, false
	for _, c := range characters {
		if c == characterId1 {
			found1 = true
		}
		if c == characterId2 {
			found2 = true
		}
	}

	if !found1 {
		t.Errorf("Character %d not found in map", characterId1)
	}
	if !found2 {
		t.Errorf("Character %d not found in map", characterId2)
	}
}

func TestProcessorImpl_Enter_Duplicate(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, te := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(3)
	channelId := channel.Id(3)
	mapId := _map.Id(100000002)
	instance := uuid.Nil
	characterId := uint32(11111)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Enter same character twice
	p.Enter(transactionId, f, characterId)
	p.Enter(transactionId, f, characterId)

	// Verify character only appears once
	key := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance}
	characters := getRegistry().GetInMap(key)

	if len(characters) != 1 {
		t.Fatalf("Expected 1 character (no duplicates), got %d", len(characters))
	}
}

func TestProcessorImpl_Exit(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, te := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(4)
	channelId := channel.Id(4)
	mapId := _map.Id(100000003)
	instance := uuid.Nil
	characterId := uint32(22222)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Enter character
	p.Enter(transactionId, f, characterId)

	// Verify character is in map
	key := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance}
	characters := getRegistry().GetInMap(key)
	if len(characters) != 1 {
		t.Fatalf("Expected 1 character after enter, got %d", len(characters))
	}

	// Exit character
	p.Exit(transactionId, f, characterId)

	// Verify character is no longer in map
	characters = getRegistry().GetInMap(key)
	if len(characters) != 0 {
		t.Fatalf("Expected 0 characters after exit, got %d", len(characters))
	}
}

func TestProcessorImpl_Exit_NotInMap(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, te := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(5)
	channelId := channel.Id(5)
	mapId := _map.Id(100000004)
	instance := uuid.Nil
	characterId := uint32(33333)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Exit character that was never in map (should not panic)
	p.Exit(transactionId, f, characterId)

	// Verify map is empty
	key := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance}
	characters := getRegistry().GetInMap(key)
	if len(characters) != 0 {
		t.Fatalf("Expected 0 characters, got %d", len(characters))
	}
}

func TestProcessorImpl_GetCharactersInMap(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, te := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(6)
	channelId := channel.Id(6)
	mapId := _map.Id(100000005)
	instance := uuid.Nil
	characterId1 := uint32(44444)
	characterId2 := uint32(55555)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Enter characters
	p.Enter(transactionId, f, characterId1)
	p.Enter(transactionId, f, characterId2)

	// Get characters via processor
	characters, err := p.GetCharactersInMap(transactionId, f)
	if err != nil {
		t.Fatalf("GetCharactersInMap returned error: %v", err)
	}

	if len(characters) != 2 {
		t.Fatalf("Expected 2 characters, got %d", len(characters))
	}

	// Verify correct characters returned
	key := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance}
	registryChars := getRegistry().GetInMap(key)
	if len(registryChars) != len(characters) {
		t.Errorf("Processor returned different count than registry: %d vs %d", len(characters), len(registryChars))
	}
}

func TestProcessorImpl_GetCharactersInMap_Empty(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, _ := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(7)
	channelId := channel.Id(7)
	mapId := _map.Id(100000006)
	instance := uuid.Nil

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Get characters from empty map
	characters, err := p.GetCharactersInMap(transactionId, f)
	if err != nil {
		t.Fatalf("GetCharactersInMap returned error: %v", err)
	}

	if characters != nil && len(characters) != 0 {
		t.Fatalf("Expected empty slice, got %d characters", len(characters))
	}
}

func TestProcessorImpl_GetMapsWithCharacters(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, te := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(8)
	channelId := channel.Id(8)
	mapId1 := _map.Id(100000007)
	mapId2 := _map.Id(100000008)
	instance := uuid.Nil
	characterId1 := uint32(66666)
	characterId2 := uint32(77777)

	f1 := field.NewBuilder(worldId, channelId, mapId1).SetInstance(instance).Build()
	f2 := field.NewBuilder(worldId, channelId, mapId2).SetInstance(instance).Build()

	// Enter characters in different maps
	p.Enter(transactionId, f1, characterId1)
	p.Enter(transactionId, f2, characterId2)

	// Get maps with characters
	maps := p.GetMapsWithCharacters()

	// Find our maps in the result
	key1 := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId1, Instance: instance}
	key2 := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId2, Instance: instance}

	foundMap1, foundMap2 := false, false
	for _, mk := range maps {
		if mk == key1 {
			foundMap1 = true
		}
		if mk == key2 {
			foundMap2 = true
		}
	}

	if !foundMap1 {
		t.Errorf("Map %v not found in GetMapsWithCharacters result", mapId1)
	}
	if !foundMap2 {
		t.Errorf("Map %v not found in GetMapsWithCharacters result", mapId2)
	}
}

func TestProcessorImpl_TenantIsolation(t *testing.T) {
	logger, _ := test.NewNullLogger()

	// Create two different tenants
	ctx1, te1 := createTestContext()
	ctx2, te2 := createTestContext()

	p1 := NewProcessor(logger, ctx1)
	p2 := NewProcessor(logger, ctx2)

	transactionId := uuid.New()
	worldId := world.Id(9)
	channelId := channel.Id(9)
	mapId := _map.Id(100000009)
	instance := uuid.Nil
	characterId1 := uint32(88888)
	characterId2 := uint32(99999)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	// Enter character in tenant 1
	p1.Enter(transactionId, f, characterId1)

	// Enter character in tenant 2
	p2.Enter(transactionId, f, characterId2)

	// Verify tenant 1 only sees their character
	chars1, _ := p1.GetCharactersInMap(transactionId, f)
	if len(chars1) != 1 {
		t.Fatalf("Tenant 1 expected 1 character, got %d", len(chars1))
	}
	if chars1[0] != characterId1 {
		t.Errorf("Tenant 1 expected character %d, got %d", characterId1, chars1[0])
	}

	// Verify tenant 2 only sees their character
	chars2, _ := p2.GetCharactersInMap(transactionId, f)
	if len(chars2) != 1 {
		t.Fatalf("Tenant 2 expected 1 character, got %d", len(chars2))
	}
	if chars2[0] != characterId2 {
		t.Errorf("Tenant 2 expected character %d, got %d", characterId2, chars2[0])
	}

	// Verify registry has separate entries
	key1 := MapKey{Tenant: te1, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance}
	key2 := MapKey{Tenant: te2, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance}

	regChars1 := getRegistry().GetInMap(key1)
	regChars2 := getRegistry().GetInMap(key2)

	if len(regChars1) != 1 || regChars1[0] != characterId1 {
		t.Errorf("Registry tenant 1 isolation failed")
	}
	if len(regChars2) != 1 || regChars2[0] != characterId2 {
		t.Errorf("Registry tenant 2 isolation failed")
	}
}

func TestProcessorImpl_InstanceIsolation(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx, te := createTestContext()

	p := NewProcessor(logger, ctx)

	transactionId := uuid.New()
	worldId := world.Id(10)
	channelId := channel.Id(10)
	mapId := _map.Id(100000010)
	instance1 := uuid.New()
	instance2 := uuid.New()
	characterId1 := uint32(11111)
	characterId2 := uint32(22222)

	f1 := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance1).Build()
	f2 := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance2).Build()
	fNil := field.NewBuilder(worldId, channelId, mapId).Build()

	// Enter character in instance 1
	p.Enter(transactionId, f1, characterId1)

	// Enter character in instance 2
	p.Enter(transactionId, f2, characterId2)

	// Verify instance 1 only sees their character
	chars1, _ := p.GetCharactersInMap(transactionId, f1)
	if len(chars1) != 1 {
		t.Fatalf("Instance 1 expected 1 character, got %d", len(chars1))
	}
	if chars1[0] != characterId1 {
		t.Errorf("Instance 1 expected character %d, got %d", characterId1, chars1[0])
	}

	// Verify instance 2 only sees their character
	chars2, _ := p.GetCharactersInMap(transactionId, f2)
	if len(chars2) != 1 {
		t.Fatalf("Instance 2 expected 1 character, got %d", len(chars2))
	}
	if chars2[0] != characterId2 {
		t.Errorf("Instance 2 expected character %d, got %d", characterId2, chars2[0])
	}

	// Verify non-instanced map has no characters
	charsNil, _ := p.GetCharactersInMap(transactionId, fNil)
	if len(charsNil) != 0 {
		t.Fatalf("Non-instanced map expected 0 characters, got %d", len(charsNil))
	}

	// Verify registry has separate entries
	key1 := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance1}
	key2 := MapKey{Tenant: te, WorldId: worldId, ChannelId: channelId, MapId: mapId, Instance: instance2}

	regChars1 := getRegistry().GetInMap(key1)
	regChars2 := getRegistry().GetInMap(key2)

	if len(regChars1) != 1 || regChars1[0] != characterId1 {
		t.Errorf("Registry instance 1 isolation failed")
	}
	if len(regChars2) != 1 || regChars2[0] != characterId2 {
		t.Errorf("Registry instance 2 isolation failed")
	}
}
