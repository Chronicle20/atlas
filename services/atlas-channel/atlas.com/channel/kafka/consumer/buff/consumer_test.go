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

// TestHandleSnapshotBuff is table-driven over the buff snapshot consumer's
// distinct scenarios (DOM-20). Each case's body is preserved verbatim from
// the original single-purpose test function, including its exact
// assertions, so no scenario's checking strength changed in the
// conversion. The two "NoSnapshot_NoOp" cases prove a negative — that an
// event for a character with no existing snapshot entry never creates
// one — and their exact predicate (BuffsGen/BuffsValid/len(Buffs), not
// merely "no error") is preserved unchanged.
func TestHandleSnapshotBuff(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "AppliedAndExpired",
			run: func(t *testing.T) {
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
			},
		},
		{
			// Confirms the registry's update-only contract holds through the
			// handler: an APPLIED event for a character with no existing
			// snapshot entry must never create one. If the handler (or the
			// registry mutator it calls) created an entry as a side effect of
			// the upsert, the entry's BuffsGen would already be bumped to 1
			// by the time the first real View() call below observes it; a
			// true no-op leaves View() to create the entry fresh, at gen 0.
			name: "Applied_NoSnapshot_NoOp",
			run: func(t *testing.T) {
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
			},
		},
		{
			// EXPIRED-side sibling of "Applied_NoSnapshot_NoOp" above.
			name: "Expired_NoSnapshot_NoOp",
			run: func(t *testing.T) {
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

// TestHandleSnapshotBuffStatUpdated proves the STAT_UPDATED arm
// (bug-shadow-buffs-dead-code part 2): a valid buffs component is
// invalidated rather than mutated in place, and an event for a character
// with no existing snapshot entry is a no-op (same update-only contract as
// APPLIED/EXPIRED).
func TestHandleSnapshotBuffStatUpdated(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "StatUpdated_InvalidatesBuffs",
			run: func(t *testing.T) {
				tm := newTestTenant(t)
				ctx := tenant.WithContext(context.Background(), tm)
				sc := newTestServer(t, tm)
				seedBuffs(t, tm, 81)

				ae := buff2.StatusEvent[buff2.AppliedStatusEventBody]{
					WorldId: 0, CharacterId: 81, Type: buff2.EventStatusTypeBuffApplied,
					Body: buff2.AppliedStatusEventBody{
						SourceId: 5220001, Level: 1, Duration: 60000,
						Changes:   []buff2.StatChange{{Type: "COMBO_COUNTER", Amount: 1}},
						CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
					},
				}
				handleSnapshotBuffApplied(sc, nil)(logrus.New(), ctx, ae)

				v := snapshot.GetRegistry().View(tm, 81)
				if !v.BuffsValid || len(v.Buffs) != 1 {
					t.Fatalf("precondition: buffs must be valid and populated: %+v", v)
				}

				su := buff2.StatusEvent[buff2.StatUpdatedStatusEventBody]{
					WorldId: 0, CharacterId: 81, Type: buff2.EventStatusTypeStatUpdated,
					Body: buff2.StatUpdatedStatusEventBody{
						SourceId: 5220001, Level: 1, Duration: 60000,
						Changes:   []buff2.StatChange{{Type: "COMBO_COUNTER", Amount: 2}},
						CreatedAt: ae.Body.CreatedAt, ExpiresAt: ae.Body.ExpiresAt,
					},
				}
				handleSnapshotBuffStatUpdated(sc, nil)(logrus.New(), ctx, su)

				v = snapshot.GetRegistry().View(tm, 81)
				if v.BuffsValid {
					t.Fatalf("STAT_UPDATED must invalidate the buffs component, not apply in place: %+v", v)
				}
			},
		},
		{
			name: "StatUpdated_NoSnapshot_NoOp",
			run: func(t *testing.T) {
				tm := newTestTenant(t)
				ctx := tenant.WithContext(context.Background(), tm)
				sc := newTestServer(t, tm)

				su := buff2.StatusEvent[buff2.StatUpdatedStatusEventBody]{
					WorldId: 0, CharacterId: 999999, Type: buff2.EventStatusTypeStatUpdated,
					Body: buff2.StatUpdatedStatusEventBody{
						SourceId: 5220001, Level: 1, Duration: 60000,
						Changes:   []buff2.StatChange{{Type: "COMBO_COUNTER", Amount: 2}},
						CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
					},
				}
				handleSnapshotBuffStatUpdated(sc, nil)(logrus.New(), ctx, su)

				v := snapshot.GetRegistry().View(tm, 999999)
				if v.BuffsGen != 0 || v.BuffsValid || len(v.Buffs) != 0 {
					t.Fatalf("STAT_UPDATED for unknown character must not create a snapshot entry: %+v", v)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
