package handler

import (
	"atlas-channel/maplelife"
	"atlas-channel/session"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newMapleLifeOpenTestSession builds a GMS v83 session for account 7,
// character 42, world 1 -- same builder shape as
// newCashItemUseTestSessionForVersion, plus SetAccountId (that helper never
// stamps one, since none of its existing callers need it).
func newMapleLifeOpenTestSession(t *testing.T, accountId uint32, characterId uint32, worldId world.Id) (session.Model, context.Context, tenant.Model, func()) {
	t.Helper()
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	sp := session.NewProcessor(logrus.New(), ctx)
	// Processor.Create is the only public path that stamps a session's
	// worldId -- session.NewSession itself takes no world, and SetField
	// (the mapId/instance-only setter every other cash-item test uses)
	// deliberately never touches worldId/channelId (session/processor.go).
	ch := channel.NewModel(worldId, channel.Id(0))
	sp.Create(ch, 0)(sessionId, discardConn{})
	sp.SetCharacterId(sessionId, characterId)
	sp.SetAccountId(sessionId, accountId)
	f := field.NewBuilder(worldId, channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, ctx, ten, func() { session.ClearRegistryForTenant(ten.Id()) }
}

func TestBeginMapleLifeRecordsPending(t *testing.T) {
	const accountId = uint32(7)
	const characterId = uint32(42)
	const worldId = world.Id(1)
	const itemId = item.Id(5431000)
	const src = slot.Position(-3)
	const updateTime = uint32(1234)

	s, ctx, ten, cleanup := newMapleLifeOpenTestSession(t, accountId, characterId, worldId)
	defer cleanup()

	beginMapleLife(logrus.New(), ctx, nil)(s, itemId, src, updateTime)

	e, ok := maplelife.GetRegistry().Get(ten, accountId)
	if !ok {
		t.Fatalf("expected a pending entry for account [%d]", accountId)
	}
	if e.CharacterId != characterId {
		t.Errorf("CharacterId = %d, want %d", e.CharacterId, characterId)
	}
	if e.WorldId != worldId {
		t.Errorf("WorldId = %d, want %d", e.WorldId, worldId)
	}
	if e.ItemId != itemId {
		t.Errorf("ItemId = %d, want %d", e.ItemId, itemId)
	}
	if e.Slot != src {
		t.Errorf("Slot = %d, want %d", e.Slot, src)
	}
	if e.UpdateTime != updateTime {
		t.Errorf("UpdateTime = %d, want %d", e.UpdateTime, updateTime)
	}
	if e.Phase != maplelife.PhaseOpen {
		t.Errorf("Phase = %q, want %q", e.Phase, maplelife.PhaseOpen)
	}
}

func TestBeginMapleLifeIsIdempotent(t *testing.T) {
	const accountId = uint32(7)
	const characterId = uint32(42)
	const worldId = world.Id(1)

	s, ctx, ten, cleanup := newMapleLifeOpenTestSession(t, accountId, characterId, worldId)
	defer cleanup()

	beginMapleLife(logrus.New(), ctx, nil)(s, item.Id(5431000), slot.Position(-3), uint32(1))
	beginMapleLife(logrus.New(), ctx, nil)(s, item.Id(5432000), slot.Position(-4), uint32(2))

	e, ok := maplelife.GetRegistry().Get(ten, accountId)
	if !ok {
		t.Fatalf("expected a pending entry for account [%d]", accountId)
	}
	if e.ItemId != item.Id(5432000) {
		t.Errorf("ItemId = %d, want the second call's value 5432000", e.ItemId)
	}
	if e.Slot != slot.Position(-4) {
		t.Errorf("Slot = %d, want the second call's value -4", e.Slot)
	}
	if e.UpdateTime != uint32(2) {
		t.Errorf("UpdateTime = %d, want the second call's value 2", e.UpdateTime)
	}
}

func TestBeginMapleLifeUsesSessionAccountAndWorld(t *testing.T) {
	const sessionAccountId = uint32(7)
	const sessionWorldId = world.Id(1)

	s, ctx, ten, cleanup := newMapleLifeOpenTestSession(t, sessionAccountId, uint32(42), sessionWorldId)
	defer cleanup()

	beginMapleLife(logrus.New(), ctx, nil)(s, item.Id(5431000), slot.Position(-3), uint32(1))

	e, ok := maplelife.GetRegistry().Get(ten, sessionAccountId)
	if !ok {
		t.Fatalf("expected a pending entry keyed on the session's account [%d]", sessionAccountId)
	}
	if e.WorldId != sessionWorldId {
		t.Errorf("WorldId = %d, want the session's world [%d]", e.WorldId, sessionWorldId)
	}
}
