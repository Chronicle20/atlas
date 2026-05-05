package character

import (
	"context"
	"testing"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	"atlas-monster-book/kafka/message"
	characterMsg "atlas-monster-book/kafka/message/character"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func tenantCtx(t *testing.T, id uuid.UUID) context.Context {
	t.Helper()
	tn, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func TestHandleDeletedCascades(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := card.Migration(db); err != nil {
		t.Fatalf("card migrate: %v", err)
	}
	if err := collection.Migration(db); err != nil {
		t.Fatalf("collection migrate: %v", err)
	}
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Seed using buffer-only Add/Recompute paths so we never need a real Kafka
	// producer (the Buffer is filled but the messages are never dispatched).
	cp := card.NewProcessor(logger, ctx, db)
	colp := collection.NewProcessor(logger, ctx, db)

	mb := message.NewBuffer()
	if _, err := cp.Add(mb)(uuid.New(), 99, 2380000); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	if err := colp.RecomputeAndEmit(mb)(99); err != nil {
		t.Fatalf("seed collection: %v", err)
	}

	// Verify seed.
	if cards, err := cp.GetByCharacterId(99); err != nil || len(cards) != 1 {
		t.Fatalf("expected 1 seeded card, got %d (err=%v)", len(cards), err)
	}
	if col, err := colp.GetByCharacterId(99); err != nil || col.NormalCount() != 1 {
		t.Fatalf("expected seeded collection NormalCount=1, got %+v (err=%v)", col, err)
	}

	handleStatusEventDeleted(db)(logger, ctx, characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]{
		CharacterId: 99,
		Type:        characterMsg.StatusEventTypeDeleted,
	})

	cards, err := cp.GetByCharacterId(99)
	if err != nil {
		t.Fatalf("get cards: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("expected cards deleted, got %d", len(cards))
	}
	col, err := colp.GetByCharacterId(99)
	if err != nil {
		t.Fatalf("get collection: %v", err)
	}
	if col.NormalCount() != 0 {
		t.Fatalf("expected collection cleared, got %+v", col)
	}
}

func TestHandleDeletedIgnoresWrongType(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := card.Migration(db); err != nil {
		t.Fatalf("card migrate: %v", err)
	}
	if err := collection.Migration(db); err != nil {
		t.Fatalf("collection migrate: %v", err)
	}
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cp := card.NewProcessor(logger, ctx, db)
	mb := message.NewBuffer()
	if _, err := cp.Add(mb)(uuid.New(), 99, 2380000); err != nil {
		t.Fatalf("seed card: %v", err)
	}

	handleStatusEventDeleted(db)(logger, ctx, characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]{
		CharacterId: 99,
		Type:        characterMsg.StatusEventTypeCreated,
	})

	cards, _ := cp.GetByCharacterId(99)
	if len(cards) != 1 {
		t.Fatalf("expected card retained on wrong type, got %d", len(cards))
	}
}
