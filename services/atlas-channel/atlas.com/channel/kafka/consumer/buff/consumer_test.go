package buff

import (
	"atlas-channel/character/snapshot"
	"atlas-channel/server"
	"context"
	"testing"
	"time"

	buff2 "atlas-channel/kafka/message/buff"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, channel.NewModel(0, 1), "127.0.0.1", 8484)
}

func seedBuffs(t *testing.T, tm tenant.Model, characterId uint32) {
	t.Helper()
	v := snapshot.GetRegistry().View(tm, characterId)
	if !snapshot.GetRegistry().BackfillBuffs(tm, characterId, nil, v.BuffsGen) {
		t.Fatalf("seed backfill rejected")
	}
}

func TestHandleSnapshotBuffAppliedAndExpired(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedBuffs(t, tm, 81)

	ae := buff2.StatusEvent[buff2.AppliedStatusEventBody]{
		WorldId: 0, CharacterId: 81, Type: buff2.EventStatusTypeBuffApplied,
		Body: buff2.AppliedStatusEventBody{
			SourceId: 3111004, Level: 20, Duration: 60000,
			Changes:   []buff2.StatChange{{Type: "SOUL_ARROW", Amount: 1}},
			CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
		},
	}
	handleSnapshotBuffApplied(sc, nil)(logrus.New(), ctx, ae)
	handleSnapshotBuffApplied(sc, nil)(logrus.New(), ctx, ae) // redelivery: no duplicate

	v := snapshot.GetRegistry().View(tm, 81)
	if len(v.Buffs) != 1 || v.Buffs[0].SourceId() != 3111004 {
		t.Fatalf("APPLIED upsert mismatch: %+v", v.Buffs)
	}
	if len(v.Buffs[0].Changes()) != 1 || v.Buffs[0].Changes()[0].Type() != "SOUL_ARROW" {
		t.Fatalf("changes not carried: %+v", v.Buffs[0].Changes())
	}

	ee := buff2.StatusEvent[buff2.ExpiredStatusEventBody]{
		WorldId: 0, CharacterId: 81, Type: buff2.EventStatusTypeBuffExpired,
		Body: buff2.ExpiredStatusEventBody{SourceId: 3111004},
	}
	handleSnapshotBuffExpired(sc, nil)(logrus.New(), ctx, ee)

	if v = snapshot.GetRegistry().View(tm, 81); len(v.Buffs) != 0 {
		t.Fatalf("EXPIRED must remove by sourceId: %+v", v.Buffs)
	}
}

// TestHandleSnapshotBuffApplied_NoSnapshot_NoOp confirms the registry's
// update-only contract holds through the handler: an APPLIED event for a
// character with no existing snapshot entry must never create one. If the
// handler (or the registry mutator it calls) created an entry as a side
// effect of the upsert, the entry's BuffsGen would already be bumped to 1
// by the time the first real View() call below observes it; a true no-op
// leaves View() to create the entry fresh, at gen 0.
func TestHandleSnapshotBuffApplied_NoSnapshot_NoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	ae := buff2.StatusEvent[buff2.AppliedStatusEventBody]{
		WorldId: 0, CharacterId: 999999, Type: buff2.EventStatusTypeBuffApplied,
		Body: buff2.AppliedStatusEventBody{
			SourceId: 3111004, Level: 20, Duration: 60000,
			Changes:   []buff2.StatChange{{Type: "SOUL_ARROW", Amount: 1}},
			CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
		},
	}
	handleSnapshotBuffApplied(sc, nil)(logrus.New(), ctx, ae)

	v := snapshot.GetRegistry().View(tm, 999999)
	if v.BuffsGen != 0 || v.BuffsValid || len(v.Buffs) != 0 {
		t.Fatalf("APPLIED for unknown character must not create a snapshot entry: %+v", v)
	}
}

// TestHandleSnapshotBuffExpired_NoSnapshot_NoOp is the EXPIRED-side sibling
// of TestHandleSnapshotBuffApplied_NoSnapshot_NoOp above.
func TestHandleSnapshotBuffExpired_NoSnapshot_NoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	ee := buff2.StatusEvent[buff2.ExpiredStatusEventBody]{
		WorldId: 0, CharacterId: 999999, Type: buff2.EventStatusTypeBuffExpired,
		Body: buff2.ExpiredStatusEventBody{SourceId: 3111004},
	}
	handleSnapshotBuffExpired(sc, nil)(logrus.New(), ctx, ee)

	v := snapshot.GetRegistry().View(tm, 999999)
	if v.BuffsGen != 0 || v.BuffsValid || len(v.Buffs) != 0 {
		t.Fatalf("EXPIRED for unknown character must not create a snapshot entry: %+v", v)
	}
}
