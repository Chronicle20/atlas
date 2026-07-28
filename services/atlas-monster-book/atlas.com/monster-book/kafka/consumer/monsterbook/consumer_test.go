package monsterbook

import (
	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	"context"
	"testing"

	mbmsg "atlas-monster-book/kafka/message/monsterbook"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func tenantCtx(t *testing.T, id uuid.UUID) context.Context {
	t.Helper()
	tn, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func TestHandleCardPickedUpInsertsAndRecomputes(t *testing.T) {
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
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("outbox migrate: %v", err)
	}
	tid := uuid.New()
	ctx := tenantCtx(t, tid)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	handleCardPickedUp(db)(logger, ctx, mbmsg.Command[mbmsg.CardPickedUpBody]{
		TenantId:    tid,
		CharacterId: 1,
		EventId:     uuid.New(),
		Type:        mbmsg.CommandTypeCardPickedUp,
		Body:        mbmsg.CardPickedUpBody{CardId: 2380000},
	})

	cp := card.NewProcessor(logger, ctx, db)
	cards, err := cp.GetByCharacterId(1)
	if err != nil {
		t.Fatalf("get cards: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}

	colp := collection.NewProcessor(logger, ctx, db)
	col, err := colp.GetByCharacterId(1)
	if err != nil {
		t.Fatalf("get collection: %v", err)
	}
	if col.NormalCount() != 1 || col.BookLevel() != 1 {
		t.Fatalf("collection wrong: NormalCount=%d BookLevel=%d", col.NormalCount(), col.BookLevel())
	}

	// The CARD_ADDED status event (buffered by card.Add) and the
	// STATS_CHANGED + EXPERIENCE_CHANGED events (buffered by
	// collection.RecomputeAndEmit) must land as outbox rows enqueued in the
	// SAME transaction as the domain writes above, not published directly.
	var outboxCount int64
	if err := db.Model(&outbox.Entity{}).Count(&outboxCount).Error; err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if outboxCount != 3 {
		t.Fatalf("expected 3 outbox rows (CARD_ADDED, STATS_CHANGED, EXPERIENCE_CHANGED), got %d", outboxCount)
	}
}

func TestHandleCardPickedUpIgnoresWrongType(t *testing.T) {
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

	handleCardPickedUp(db)(logger, ctx, mbmsg.Command[mbmsg.CardPickedUpBody]{
		TenantId:    tid,
		CharacterId: 1,
		EventId:     uuid.New(),
		Type:        "OTHER",
		Body:        mbmsg.CardPickedUpBody{CardId: 2380000},
	})

	cp := card.NewProcessor(logger, ctx, db)
	cards, _ := cp.GetByCharacterId(1)
	if len(cards) != 0 {
		t.Fatalf("expected no cards inserted, got %d", len(cards))
	}
}
