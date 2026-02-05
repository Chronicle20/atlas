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
	"github.com/sirupsen/logrus"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func testContext() context.Context {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}

func TestProcessor_Enter(t *testing.T) {
	l := testLogger()
	ctx := testContext()
	p := NewProcessor(l, ctx)
	r := getRegistry()

	characterId := uint32(20001)
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(100000000)

	// Clean up
	r.RemoveCharacter(characterId)

	f := field.NewBuilder(worldId, channelId, mapId).Build()
	p.Enter(f, characterId)

	mk, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should be registered after Enter")
	}

	if mk.Field.WorldId() != worldId {
		t.Errorf("expected WorldId %d, got %d", worldId, mk.Field.WorldId())
	}
	if mk.Field.ChannelId() != channelId {
		t.Errorf("expected ChannelId %d, got %d", channelId, mk.Field.ChannelId())
	}
	if mk.Field.MapId() != mapId {
		t.Errorf("expected MapId %d, got %d", mapId, mk.Field.MapId())
	}

	// Clean up
	r.RemoveCharacter(characterId)
}

func TestProcessor_Exit(t *testing.T) {
	l := testLogger()
	ctx := testContext()
	p := NewProcessor(l, ctx)
	r := getRegistry()

	characterId := uint32(20002)
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(100000000)

	// Setup: enter first
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	p.Enter(f, characterId)

	// Verify character exists
	_, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should exist before Exit")
	}

	// Exit
	p.Exit(f, characterId)

	// Verify character is gone
	_, ok = r.GetMap(characterId)
	if ok {
		t.Error("character should not exist after Exit")
	}
}

func TestProcessor_GetMap(t *testing.T) {
	l := testLogger()
	ctx := testContext()
	p := NewProcessor(l, ctx)
	r := getRegistry()

	characterId := uint32(20003)
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(100000000)

	// Clean up
	r.RemoveCharacter(characterId)

	// Enter
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	p.Enter(f, characterId)

	// Get map via processor
	m, err := p.GetMap(characterId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.WorldId() != worldId {
		t.Errorf("expected WorldId %d, got %d", worldId, m.WorldId())
	}
	if m.ChannelId() != channelId {
		t.Errorf("expected ChannelId %d, got %d", channelId, m.ChannelId())
	}
	if m.MapId() != mapId {
		t.Errorf("expected MapId %d, got %d", mapId, m.MapId())
	}

	// Clean up
	r.RemoveCharacter(characterId)
}

func TestProcessor_GetMap_NotFound(t *testing.T) {
	l := testLogger()
	ctx := testContext()
	p := NewProcessor(l, ctx)
	r := getRegistry()

	characterId := uint32(99999998)

	// Ensure doesn't exist
	r.RemoveCharacter(characterId)

	_, err := p.GetMap(characterId)
	if err == nil {
		t.Error("expected error for non-existent character")
	}
}

func TestProcessor_TransitionMap(t *testing.T) {
	l := testLogger()
	ctx := testContext()
	p := NewProcessor(l, ctx)
	r := getRegistry()

	characterId := uint32(20004)
	worldId := world.Id(1)
	channelId := channel.Id(2)
	oldMapId := _map.Id(100000000)
	newMapId := _map.Id(200000000)

	// Clean up
	r.RemoveCharacter(characterId)

	// Enter old map
	f := field.NewBuilder(worldId, channelId, oldMapId).Build()
	p.Enter(f, characterId)

	// Transition to new map
	f = f.Clone().SetMapId(newMapId).Build()
	p.TransitionMap(f, characterId)

	// Verify new location
	mk, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should exist after transition")
	}

	if mk.Field.MapId() != newMapId {
		t.Errorf("expected MapId %d, got %d", newMapId, mk.Field.MapId())
	}

	// Clean up
	r.RemoveCharacter(characterId)
}

func TestProcessor_TransitionChannel(t *testing.T) {
	l := testLogger()
	ctx := testContext()
	p := NewProcessor(l, ctx)
	r := getRegistry()

	characterId := uint32(20005)
	worldId := world.Id(1)
	oldChannelId := channel.Id(1)
	newChannelId := channel.Id(2)
	mapId := _map.Id(100000000)

	// Clean up
	r.RemoveCharacter(characterId)

	// Enter with old channel
	f := field.NewBuilder(worldId, oldChannelId, mapId).Build()
	p.Enter(f, characterId)

	// Transition channel
	f = f.Clone().SetChannelId(newChannelId).Build()
	p.TransitionChannel(f, characterId)

	// Verify new channel
	mk, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should exist after channel transition")
	}

	if mk.Field.ChannelId() != newChannelId {
		t.Errorf("expected ChannelId %d, got %d", newChannelId, mk.Field.ChannelId())
	}

	// Clean up
	r.RemoveCharacter(characterId)
}
