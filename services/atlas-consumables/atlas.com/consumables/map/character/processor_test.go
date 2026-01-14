package character

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
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
	worldId := byte(1)
	channelId := byte(2)
	mapId := uint32(100000000)

	// Clean up
	r.RemoveCharacter(characterId)

	p.Enter(worldId, channelId, mapId, characterId)

	mk, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should be registered after Enter")
	}

	if mk.WorldId != worldId {
		t.Errorf("expected WorldId %d, got %d", worldId, mk.WorldId)
	}
	if mk.ChannelId != channelId {
		t.Errorf("expected ChannelId %d, got %d", channelId, mk.ChannelId)
	}
	if mk.MapId != mapId {
		t.Errorf("expected MapId %d, got %d", mapId, mk.MapId)
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
	worldId := byte(1)
	channelId := byte(2)
	mapId := uint32(100000000)

	// Setup: enter first
	p.Enter(worldId, channelId, mapId, characterId)

	// Verify character exists
	_, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should exist before Exit")
	}

	// Exit
	p.Exit(worldId, channelId, mapId, characterId)

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
	worldId := byte(1)
	channelId := byte(2)
	mapId := uint32(100000000)

	// Clean up
	r.RemoveCharacter(characterId)

	// Enter
	p.Enter(worldId, channelId, mapId, characterId)

	// Get map via processor
	m, err := p.GetMap(characterId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.WorldId() != world.Id(worldId) {
		t.Errorf("expected WorldId %d, got %d", worldId, m.WorldId())
	}
	if m.ChannelId() != channel.Id(channelId) {
		t.Errorf("expected ChannelId %d, got %d", channelId, m.ChannelId())
	}
	if uint32(m.MapId()) != mapId {
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
	worldId := byte(1)
	channelId := byte(2)
	oldMapId := uint32(100000000)
	newMapId := uint32(200000000)

	// Clean up
	r.RemoveCharacter(characterId)

	// Enter old map
	p.Enter(worldId, channelId, oldMapId, characterId)

	// Transition to new map
	p.TransitionMap(worldId, channelId, newMapId, characterId, oldMapId)

	// Verify new location
	mk, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should exist after transition")
	}

	if mk.MapId != newMapId {
		t.Errorf("expected MapId %d, got %d", newMapId, mk.MapId)
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
	worldId := byte(1)
	oldChannelId := byte(1)
	newChannelId := byte(2)
	mapId := uint32(100000000)

	// Clean up
	r.RemoveCharacter(characterId)

	// Enter with old channel
	p.Enter(worldId, oldChannelId, mapId, characterId)

	// Transition channel
	p.TransitionChannel(worldId, newChannelId, oldChannelId, characterId, mapId)

	// Verify new channel
	mk, ok := r.GetMap(characterId)
	if !ok {
		t.Fatal("character should exist after channel transition")
	}

	if mk.ChannelId != newChannelId {
		t.Errorf("expected ChannelId %d, got %d", newChannelId, mk.ChannelId)
	}

	// Clean up
	r.RemoveCharacter(characterId)
}
