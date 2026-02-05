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

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testField(worldId world.Id, channelId channel.Id, mapId _map.Id) field.Model {
	return field.NewBuilder(worldId, channelId, mapId).Build()
}

func resetProcessorRegistry() {
	r := getRegistry()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister = make(map[MapKey][]uint32)
}

func TestInMapProvider(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	f := testField(0, 1, 100000000)

	// Add characters directly to registry
	key := MapKey{Tenant: st, Field: field.NewBuilder(0, 1, 100000000).Build()}
	getRegistry().AddCharacter(key, 100)
	getRegistry().AddCharacter(key, 200)

	// Get via processor
	p := NewProcessor(l, tctx)
	chars, err := p.InMapProvider(f)()
	if err != nil {
		t.Fatalf("InMapProvider failed: %v", err)
	}

	if len(chars) != 2 {
		t.Fatalf("Expected 2 characters, got %d", len(chars))
	}
}

func TestGetCharactersInMap(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	f := testField(0, 1, 100000000)

	// Add characters
	key := MapKey{Tenant: st, Field: field.NewBuilder(0, 1, 100000000).Build()}
	getRegistry().AddCharacter(key, 100)

	p := NewProcessor(l, tctx)
	chars, err := p.GetCharactersInMap(f)
	if err != nil {
		t.Fatalf("GetCharactersInMap failed: %v", err)
	}

	if len(chars) != 1 || chars[0] != 100 {
		t.Errorf("Expected [100], got %v", chars)
	}
}

func TestEnter(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	f := testField(0, 1, 100000000)
	characterId := uint32(12345)

	p := NewProcessor(l, tctx)
	p.Enter(f, characterId)

	chars, err := p.GetCharactersInMap(f)
	if err != nil {
		t.Fatalf("GetCharactersInMap failed: %v", err)
	}

	if len(chars) != 1 {
		t.Fatalf("Expected 1 character after Enter, got %d", len(chars))
	}

	if chars[0] != characterId {
		t.Errorf("Expected character %d, got %d", characterId, chars[0])
	}
}

func TestExit(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	f := testField(0, 1, 100000000)
	characterId := uint32(12345)

	p := NewProcessor(l, tctx)

	// Enter first
	p.Enter(f, characterId)

	chars, _ := p.GetCharactersInMap(f)
	if len(chars) != 1 {
		t.Fatalf("Expected 1 character after Enter, got %d", len(chars))
	}

	// Exit
	p.Exit(f, characterId)

	chars, _ = p.GetCharactersInMap(f)
	if len(chars) != 0 {
		t.Fatalf("Expected 0 characters after Exit, got %d", len(chars))
	}
}

func TestTransitionMap(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	oldField := testField(0, 1, 100000000)
	newField := testField(0, 1, 200000000)
	characterId := uint32(12345)

	p := NewProcessor(l, tctx)

	// Enter old map
	p.Enter(oldField, characterId)

	// Verify in old map
	chars, _ := p.GetCharactersInMap(oldField)
	if len(chars) != 1 {
		t.Fatalf("Expected 1 character in old map, got %d", len(chars))
	}

	// Transition to new map
	p.TransitionMap(oldField, newField, characterId)

	// Verify not in old map
	chars, _ = p.GetCharactersInMap(oldField)
	if len(chars) != 0 {
		t.Errorf("Expected 0 characters in old map after transition, got %d", len(chars))
	}

	// Verify in new map
	chars, _ = p.GetCharactersInMap(newField)
	if len(chars) != 1 {
		t.Errorf("Expected 1 character in new map after transition, got %d", len(chars))
	}

	if chars[0] != characterId {
		t.Errorf("Expected character %d in new map, got %d", characterId, chars[0])
	}
}

func TestTransitionChannel(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	oldField := testField(0, 1, 100000000)
	newField := testField(0, 2, 100000000)
	characterId := uint32(12345)

	p := NewProcessor(l, tctx)

	// Enter old channel
	p.Enter(oldField, characterId)

	// Verify in old channel
	chars, _ := p.GetCharactersInMap(oldField)
	if len(chars) != 1 {
		t.Fatalf("Expected 1 character in old channel, got %d", len(chars))
	}

	// Transition to new channel
	p.TransitionChannel(oldField, newField, characterId)

	// Verify not in old channel
	chars, _ = p.GetCharactersInMap(oldField)
	if len(chars) != 0 {
		t.Errorf("Expected 0 characters in old channel after transition, got %d", len(chars))
	}

	// Verify in new channel
	chars, _ = p.GetCharactersInMap(newField)
	if len(chars) != 1 {
		t.Errorf("Expected 1 character in new channel after transition, got %d", len(chars))
	}
}

func TestTenantIsolation(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()

	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	tctx1 := tenant.WithContext(context.Background(), tenant1)
	tctx2 := tenant.WithContext(context.Background(), tenant2)

	f := testField(0, 1, 100000000)

	p1 := NewProcessor(l, tctx1)
	p2 := NewProcessor(l, tctx2)

	// Enter same map with different tenants
	p1.Enter(f, 100)
	p2.Enter(f, 200)

	// Verify tenant1 only sees its character
	chars1, _ := p1.GetCharactersInMap(f)
	if len(chars1) != 1 || chars1[0] != 100 {
		t.Errorf("Tenant1 expected [100], got %v", chars1)
	}

	// Verify tenant2 only sees its character
	chars2, _ := p2.GetCharactersInMap(f)
	if len(chars2) != 1 || chars2[0] != 200 {
		t.Errorf("Tenant2 expected [200], got %v", chars2)
	}
}

func TestMultipleCharactersInMap(t *testing.T) {
	resetProcessorRegistry()
	l, _ := test.NewNullLogger()
	st := testTenant()
	tctx := tenant.WithContext(context.Background(), st)

	f := testField(0, 1, 100000000)

	p := NewProcessor(l, tctx)

	// Enter multiple characters
	p.Enter(f, 100)
	p.Enter(f, 200)
	p.Enter(f, 300)

	chars, _ := p.GetCharactersInMap(f)
	if len(chars) != 3 {
		t.Fatalf("Expected 3 characters, got %d", len(chars))
	}

	// Exit one character
	p.Exit(f, 200)

	chars, _ = p.GetCharactersInMap(f)
	if len(chars) != 2 {
		t.Fatalf("Expected 2 characters after exit, got %d", len(chars))
	}

	// Verify correct characters remain
	charMap := make(map[uint32]bool)
	for _, c := range chars {
		charMap[c] = true
	}

	if !charMap[100] || !charMap[300] {
		t.Errorf("Expected characters 100 and 300 to remain, got %v", chars)
	}

	if charMap[200] {
		t.Error("Character 200 should have been removed")
	}
}
