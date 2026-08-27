package skill

import (
	"atlas-channel/character/snapshot"
	"atlas-channel/server"
	"context"
	"testing"
	"time"

	skillmodel "atlas-channel/character/skill"

	skill2 "atlas-channel/kafka/message/skill"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
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

func seedSkills(t *testing.T, tm tenant.Model, characterId uint32, ms []skillmodel.Model) {
	t.Helper()
	v := snapshot.GetRegistry().View(tm, characterId)
	if !snapshot.GetRegistry().BackfillSkills(tm, characterId, ms, v.SkillsGen) {
		t.Fatalf("seed backfill rejected")
	}
}

func TestHandleSnapshotSkillCreatedAndUpdated_Upsert(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSkills(t, tm, 51, nil)

	ce := skill2.StatusEvent[skill2.StatusEventCreatedBody]{
		CharacterId: 51, SkillId: 3121004, Type: skill2.StatusEventTypeCreated,
		Body: skill2.StatusEventCreatedBody{Level: 1, MasterLevel: 30, Expiration: time.Time{}},
	}
	handleSnapshotSkillCreated(sc, nil)(logrus.New(), ctx, ce)

	v := snapshot.GetRegistry().View(tm, 51)
	if len(v.Skills) != 1 || v.Skills[0].Id() != skillconst.Id(3121004) || v.Skills[0].Level() != 1 {
		t.Fatalf("CREATED upsert mismatch: %+v", v.Skills)
	}

	ue := skill2.StatusEvent[skill2.StatusEventUpdatedBody]{
		CharacterId: 51, SkillId: 3121004, Type: skill2.StatusEventTypeUpdated,
		Body: skill2.StatusEventUpdatedBody{Level: 2, MasterLevel: 30, Expiration: time.Time{}},
	}
	handleSnapshotSkillUpdated(sc, nil)(logrus.New(), ctx, ue)

	v = snapshot.GetRegistry().View(tm, 51)
	if len(v.Skills) != 1 || v.Skills[0].Level() != 2 {
		t.Fatalf("UPDATED upsert mismatch: %+v", v.Skills)
	}
}

func TestHandleSnapshotSkillDeleted_Removes(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSkills(t, tm, 52, []skillmodel.Model{
		skillmodel.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild(),
	})

	de := skill2.StatusEvent[skill2.StatusEventDeletedBody]{
		CharacterId: 52, SkillId: 3121004, Type: skill2.StatusEventTypeDeleted,
	}
	handleSnapshotSkillDeleted(sc, nil)(logrus.New(), ctx, de)

	if v := snapshot.GetRegistry().View(tm, 52); len(v.Skills) != 0 {
		t.Fatalf("DELETED must remove the skill: %+v", v.Skills)
	}
}

// TestHandleSnapshotSkillDeleted_NoSnapshot_NoOp confirms the registry's
// update-only contract holds through the handler: a DELETED event for a
// character with no existing snapshot entry must never create one. If the
// handler (or the registry mutator it calls) created an entry as a side
// effect of the delete, the entry's SkillsGen would already be bumped to 1
// by the time the first real View() call below observes it; a true no-op
// leaves View() to create the entry fresh, at gen 0.
func TestHandleSnapshotSkillDeleted_NoSnapshot_NoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	de := skill2.StatusEvent[skill2.StatusEventDeletedBody]{
		CharacterId: 999999, SkillId: 3121004, Type: skill2.StatusEventTypeDeleted,
	}
	handleSnapshotSkillDeleted(sc, nil)(logrus.New(), ctx, de)

	v := snapshot.GetRegistry().View(tm, 999999)
	if v.SkillsGen != 0 || v.SkillsValid || len(v.Skills) != 0 {
		t.Fatalf("DELETED for unknown character must not create a snapshot entry: %+v", v)
	}
}
