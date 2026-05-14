package character

import (
	"context"
	"testing"

	"atlas-maps/character/location"
	characterKafka "atlas-maps/kafka/message/character"
	mapcharacter "atlas-maps/map/character"
	"atlas-maps/visit"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
