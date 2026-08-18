package cashshop

import (
	"atlas-maps/character/location"
	"context"
	"testing"

	cashshopKafka "atlas-maps/kafka/message/cashshop"

	"github.com/google/uuid"
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

// seedOnline puts a live row on channel 5 in the given state.
func seedOnline(t *testing.T, ctx context.Context, db *gorm.DB, characterId uint32, state characterconst.PresenceState) {
	t.Helper()
	logger, _ := test.NewNullLogger()
	f := field.NewBuilder(world.Id(1), channel.Id(5), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	lp := location.NewProcessor(logger, ctx, db)
	if _, err := lp.Set(characterId, f); err != nil {
		t.Fatalf("seed Set: %v", err)
	}
	if err := lp.SetState(characterId, state); err != nil {
		t.Fatalf("seed SetState: %v", err)
	}
}

func stateOf(t *testing.T, ctx context.Context, db *gorm.DB, characterId uint32) characterconst.PresenceState {
	t.Helper()
	logger, _ := test.NewNullLogger()
	m, err := location.NewProcessor(logger, ctx, db).GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	return m.State()
}

func enterEvent(characterId uint32) cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody] {
	return cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody]{
		WorldId: world.Id(1),
		Type:    cashshopKafka.EventCashShopStatusTypeCharacterEnter,
		Body: cashshopKafka.CharacterMovementBody{
			CharacterId: characterId,
			ChannelId:   channel.Id(5),
			MapId:       _map.Id(100000000),
		},
	}
}

func exitEvent(characterId uint32) cashshopKafka.StatusEvent[cashshopKafka.CharacterMovementBody] {
	e := enterEvent(characterId)
	e.Type = cashshopKafka.EventCashShopStatusTypeCharacterExit
	return e
}

func TestEnterHandler_SetsInCashShop(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 42
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateInField)

	handleStatusEventEnterFunc(db)(logger, ctx, enterEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateInCashShop {
		t.Errorf("state = %q, want IN_CASH_SHOP", got)
	}
}

// Kafka delivery is at-least-once; a replayed ENTER must be a no-op.
func TestEnterHandler_IsIdempotent(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 43
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateInField)

	handleStatusEventEnterFunc(db)(logger, ctx, enterEvent(characterId))
	handleStatusEventEnterFunc(db)(logger, ctx, enterEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateInCashShop {
		t.Errorf("state after replay = %q, want IN_CASH_SHOP", got)
	}
}

func TestExitHandler_SetsInField(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 44
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateInCashShop)

	handleStatusEventExitFunc(db)(logger, ctx, exitEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateInField {
		t.Errorf("state = %q, want IN_FIELD", got)
	}
}

// Design §1.3: OFFLINE is terminal except via LOGIN / CHANNEL_CHANGED.
// Disconnecting from inside the cash shop emits LOGOUT and no CHARACTER_EXIT,
// so an EXIT arriving after a LOGOUT is exactly the late-delivery case.
func TestExitHandler_DoesNotResurrectOfflineCharacter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 45
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateOffline)

	handleStatusEventExitFunc(db)(logger, ctx, exitEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateOffline {
		t.Errorf("late CHARACTER_EXIT resurrected the row to %q, want OFFLINE", got)
	}
}

func TestEnterHandler_DoesNotResurrectOfflineCharacter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	db := newTestDB(t)

	const characterId uint32 = 46
	seedOnline(t, ctx, db, characterId, characterconst.PresenceStateOffline)

	handleStatusEventEnterFunc(db)(logger, ctx, enterEvent(characterId))

	if got := stateOf(t, ctx, db, characterId); got != characterconst.PresenceStateOffline {
		t.Errorf("late CHARACTER_ENTER resurrected the row to %q, want OFFLINE", got)
	}
}
