package character

import (
	"atlas-maps/character/location"
	"atlas-maps/visit"
	"context"
	"testing"

	characterKafka "atlas-maps/kafka/message/character"
	mapcharacter "atlas-maps/map/character"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := location.Migration(db); err != nil {
		t.Fatalf("location.Migration: %v", err)
	}
	if err := visit.MigrateTable(db); err != nil {
		t.Fatalf("visit.MigrateTable: %v", err)
	}
	return db
}

func newTestCtx(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

// TestDeletedHandler_RemovesLocationRow verifies that handling a DELETED event
// removes the character_locations row and drops the in-memory map registry entry.
func TestDeletedHandler_RemovesLocationRow(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 42

	// Seed a location row for the character.
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	lp := location.NewProcessor(logrus.New(), ctx, db)
	if _, err := lp.Set(characterId, f); err != nil {
		t.Fatalf("location.Set: %v", err)
	}

	// Verify row is present before deletion.
	if _, err := lp.GetById(characterId); err != nil {
		t.Fatalf("location.GetById (pre-delete): %v", err)
	}

	// Seed the in-memory registry.
	cp := mapcharacter.NewProcessor(logger, ctx)
	cp.Enter(uuid.New(), f, characterId)
	chars, _ := cp.GetCharactersInMap(uuid.New(), f)
	if len(chars) != 1 {
		t.Fatalf("expected 1 character in registry before deletion, got %d", len(chars))
	}

	// Fire the DELETED handler.
	handler := handleStatusEventDeletedFunc(logger, db)
	event := characterKafka.StatusEvent[characterKafka.StatusEventDeletedBody]{
		Type:        characterKafka.EventCharacterStatusTypeDeleted,
		CharacterId: characterId,
	}
	handler(logger, ctx, event)

	// Verify the location row is gone.
	if _, err := lp.GetById(characterId); err == nil {
		t.Error("expected location row to be deleted, but GetById returned no error")
	}

	// Verify the in-memory registry entry is gone.
	chars, _ = cp.GetCharactersInMap(uuid.New(), f)
	if len(chars) != 0 {
		t.Errorf("expected 0 characters in registry after deletion, got %d", len(chars))
	}
}

// TestDeletedHandler_IdempotentWithNoRow verifies that handling a DELETED event
// for a character with no location row does not return an error (idempotent).
func TestDeletedHandler_IdempotentWithNoRow(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 999

	// Ensure no row exists.
	lp := location.NewProcessor(logrus.New(), ctx, db)
	if _, err := lp.GetById(characterId); err == nil {
		t.Fatal("precondition: row should not exist before test")
	}

	// Fire the DELETED handler — must not panic or produce a fatal error.
	handler := handleStatusEventDeletedFunc(logger, db)
	event := characterKafka.StatusEvent[characterKafka.StatusEventDeletedBody]{
		Type:        characterKafka.EventCharacterStatusTypeDeleted,
		CharacterId: characterId,
	}
	// Should complete without panicking.
	handler(logger, ctx, event)
}

// TestLoginHandler_SetsInField — LOGIN is one of the two events that
// legitimately mean "this character is live right now", so it asserts liveness
// unconditionally.
func TestLoginHandler_SetsInField(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 42
	event := characterKafka.StatusEvent[characterKafka.StatusEventLoginBody]{
		CharacterId: characterId,
		WorldId:     world.Id(1),
		Type:        characterKafka.EventCharacterStatusTypeLogin,
		Body: characterKafka.StatusEventLoginBody{
			ChannelId: channel.Id(7),
			MapId:     _map.Id(100000000),
			Instance:  uuid.Nil,
		},
	}
	handleStatusEventLoginFunc(db)(logger, ctx, event)

	m, err := location.NewProcessor(logger, ctx, db).GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateInField {
		t.Errorf("state = %q, want IN_FIELD", m.State())
	}
	if m.ChannelId() != channel.Id(7) {
		t.Errorf("channel = %d, want 7", m.ChannelId())
	}
}

// TestLogoutHandler_SetsOfflineAndPreservesPosition — LOGOUT persists the
// last-known position (so the next login can restore it) AND marks the
// character offline. Both halves matter: /find must stop reporting a channel,
// but the login path still needs the map.
func TestLogoutHandler_SetsOfflineAndPreservesPosition(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 43
	seed := field.NewBuilder(world.Id(1), channel.Id(4), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	lp := location.NewProcessor(logger, ctx, db)
	if _, err := lp.Set(characterId, seed); err != nil {
		t.Fatalf("seed Set: %v", err)
	}
	if err := lp.SetState(characterId, characterconst.PresenceStateInField); err != nil {
		t.Fatalf("seed SetState: %v", err)
	}

	event := characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]{
		CharacterId: characterId,
		WorldId:     world.Id(1),
		Type:        characterKafka.EventCharacterStatusTypeLogout,
		Body: characterKafka.StatusEventLogoutBody{
			ChannelId: channel.Id(4),
			MapId:     _map.Id(100000000),
			Instance:  uuid.Nil,
		},
	}
	handleStatusEventLogoutFunc(db)(logger, ctx, event)

	m, err := lp.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateOffline {
		t.Errorf("state = %q, want OFFLINE", m.State())
	}
	if m.MapId() == 0 {
		t.Error("LOGOUT discarded the position; the login path needs it")
	}
}

// TestChannelChangedHandler_SetsInFieldOnNewChannel — the other event that
// legitimately asserts liveness. Channel 7 is deliberately neither 0 nor 1, so
// a handler that writes a constant cannot pass.
func TestChannelChangedHandler_SetsInFieldOnNewChannel(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 44
	event := characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]{
		CharacterId: characterId,
		WorldId:     world.Id(1),
		Type:        characterKafka.EventCharacterStatusTypeChannelChanged,
		Body: characterKafka.ChangeChannelEventLoginBody{
			ChannelId:    channel.Id(7),
			OldChannelId: channel.Id(2),
			MapId:        _map.Id(100000000),
			Instance:     uuid.Nil,
		},
	}
	handleStatusEventChannelChangedFunc(db)(logger, ctx, event)

	m, err := location.NewProcessor(logger, ctx, db).GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateInField {
		t.Errorf("state = %q, want IN_FIELD", m.State())
	}
	if m.ChannelId() != channel.Id(7) {
		t.Errorf("channel = %d, want 7", m.ChannelId())
	}
}
