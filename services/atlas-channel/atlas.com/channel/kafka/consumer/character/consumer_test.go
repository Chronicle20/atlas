package character

import (
	"atlas-channel/character/snapshot"
	"atlas-channel/server"
	"context"
	"testing"

	character3 "atlas-channel/character"

	character2 "atlas-channel/kafka/message/character"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
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
	ch := channel.NewModel(0, 1)
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// seedSnapshotCore creates a snapshot entry and validates its core so
// in-place event updates have a base to apply to.
func seedSnapshotCore(t *testing.T, tm tenant.Model, characterId uint32) {
	t.Helper()
	v := snapshot.GetRegistry().View(tm, characterId)
	core := character3.NewModelBuilder().
		SetId(characterId).SetLevel(30).SetMp(500).SetMaxMp(800).
		MustBuild()
	if !snapshot.GetRegistry().BackfillCore(tm, characterId, core, v.CoreGen) {
		t.Fatalf("seed backfill rejected")
	}
}

func TestHandleSnapshotStatChanged_RichValuesApplyInPlace(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 41)

	e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId: 0, CharacterId: 41, Type: character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{
			ChannelId: 1,
			Updates:   []stat.Type{stat.TypeMp},
			Values:    map[string]interface{}{"mp": float64(463)},
		},
	}
	handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 41)
	if !v.CoreValid || v.Core.Mp() != 463 {
		t.Fatalf("rich STAT_CHANGED must apply in place: valid=%v mp=%d", v.CoreValid, v.Core.Mp())
	}
}

func TestHandleSnapshotStatChanged_NilValuesInvalidates(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 42)

	e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId: 0, CharacterId: 42, Type: character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{ChannelId: 1, Updates: []stat.Type{stat.TypeMp}},
	}
	handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)

	if v := snapshot.GetRegistry().View(tm, 42); v.CoreValid {
		t.Fatalf("nil-Values STAT_CHANGED must invalidate (rollout safety)")
	}
}

func TestHandleSnapshotLevelAndExperience_ApplyAbsolute(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 43)

	le := character2.StatusEvent[character2.LevelChangedStatusEventBody]{
		WorldId: 0, CharacterId: 43, Type: character2.StatusEventTypeLevelChanged,
		Body: character2.LevelChangedStatusEventBody{ChannelId: 1, Amount: 1, Current: 31},
	}
	handleSnapshotLevelChanged(sc, nil)(logrus.New(), ctx, le)

	ee := character2.StatusEvent[character2.ExperienceChangedStatusEventBody]{
		WorldId: 0, CharacterId: 43, Type: character2.StatusEventTypeExperienceChanged,
		Body: character2.ExperienceChangedStatusEventBody{ChannelId: 1, Current: 999},
	}
	handleSnapshotExperienceChanged(sc, nil)(logrus.New(), ctx, ee)

	v := snapshot.GetRegistry().View(tm, 43)
	if v.Core.Level() != 31 || v.Core.Experience() != 999 {
		t.Fatalf("level/exp not applied: %d/%d", v.Core.Level(), v.Core.Experience())
	}
}

func TestHandleSnapshotMapChanged_TargetPositionSetsOverlay(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 44)

	e := character2.StatusEvent[character2.StatusEventMapChangedBody]{
		WorldId: 0, CharacterId: 44, Type: character2.StatusEventTypeMapChanged,
		Body: character2.StatusEventMapChangedBody{
			ChannelId: 1, TargetMapId: 100000000,
			UseTargetPosition: true, TargetX: 77, TargetY: -88,
		},
	}
	handleSnapshotMapChanged(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 44)
	if !v.PosValid || v.PosX != 77 || v.PosY != -88 {
		t.Fatalf("UseTargetPosition must set the overlay: %+v", v)
	}
	if !v.CoreValid {
		t.Fatalf("UseTargetPosition path must not invalidate core (overlay covers X/Y)")
	}
}

func TestHandleSnapshotMapChanged_PortalWarpInvalidatesPositionAndCore(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 45)
	snapshot.GetRegistry().SetPosition(tm, 45, 1, 2)

	e := character2.StatusEvent[character2.StatusEventMapChangedBody]{
		WorldId: 0, CharacterId: 45, Type: character2.StatusEventTypeMapChanged,
		Body: character2.StatusEventMapChangedBody{ChannelId: 1, TargetMapId: 100000000},
	}
	handleSnapshotMapChanged(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 45)
	if v.PosValid {
		t.Fatalf("portal warp must invalidate the position overlay")
	}
	if v.CoreValid {
		t.Fatalf("portal warp must invalidate core so the next read refetches fresh REST X/Y (design §10.4)")
	}
}

func TestSnapshotHandlers_IgnoreOtherWorlds(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm) // world 0
	seedSnapshotCore(t, tm, 46)

	e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId: 3, CharacterId: 46, Type: character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{ChannelId: 1, Updates: []stat.Type{stat.TypeMp}},
	}
	handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)
	if v := snapshot.GetRegistry().View(tm, 46); !v.CoreValid {
		t.Fatalf("other-world events must be ignored")
	}
}
