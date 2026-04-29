package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"atlas-channel/monster"
	"atlas-channel/server"
	"context"
	"testing"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	ch := channelconst.NewModel(0, 1)
	return server.Register(tm, ch, "127.0.0.1", 8484)
}

func TestHandleNextSkillDecided_PutsIntoInbox(t *testing.T) {
	monster.InitNextSkillInbox()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	sc := newTestServer(t, tm)
	h := handleStatusEventNextSkillDecided(sc, nil)
	h(logrus.New(), ctx, monster2.StatusEvent[monster2.StatusEventNextSkillDecidedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		UniqueId:  42,
		Type:      monster2.EventStatusNextSkillDecided,
		Body: monster2.StatusEventNextSkillDecidedBody{
			SkillId:     100,
			SkillLevel:  1,
			DecidedAtMs: 12345,
		},
	})

	d, ok := monster.GetNextSkillInbox().TakeAndClear(tm, 42)
	if !ok || d.SkillId != 100 {
		t.Fatalf("expected inbox to have decision; got ok=%v skill=%d", ok, d.SkillId)
	}
}

func TestHandleNextSkillDecided_WrongType_DoesNotPut(t *testing.T) {
	monster.InitNextSkillInbox()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	sc := newTestServer(t, tm)
	h := handleStatusEventNextSkillDecided(sc, nil)
	h(logrus.New(), ctx, monster2.StatusEvent[monster2.StatusEventNextSkillDecidedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		UniqueId:  99,
		Type:      "WRONG_TYPE",
		Body: monster2.StatusEventNextSkillDecidedBody{
			SkillId: 100,
		},
	})

	_, ok := monster.GetNextSkillInbox().TakeAndClear(tm, 99)
	if ok {
		t.Fatalf("expected no entry for wrong event type")
	}
}

// TestHandleStatusEffectApplied_PopulatesStatusMirror verifies that a
// STATUS_APPLIED event carrying a PHYSICAL reflect window is mirrored
// into the in-process StatusMirror so that GetReflect returns the
// reflect info. This is the regression target for Task 11 — guards
// against the wire body / mirror body fields drifting apart and
// against the handler skipping the mirror call. Uses synthetic per-
// test uniqueIds for singleton isolation since the mirror is process-
// wide and self-initialised via sync.Once.
func TestHandleStatusEffectApplied_PopulatesStatusMirror(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	uniqueId := uint32(424242)
	defer monster.GetStatusMirror().OnMonsterGone(tm, uniqueId)

	h := handleStatusEffectApplied(sc, nil)
	h(logrus.New(), ctx, monster2.StatusEvent[monster2.StatusEffectAppliedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		UniqueId:  uniqueId,
		Type:      monster2.EventStatusEffectApplied,
		Body: monster2.StatusEffectAppliedBody{
			EffectId:          uuid.NewString(),
			SourceType:        "CHARACTER",
			SourceCharacterId: 99,
			SourceSkillId:     1311006,
			SourceSkillLevel:  1,
			Statuses:          map[string]int32{"WEAPON_REFLECT": 1},
			Duration:          60000,
			ReflectKind:       "PHYSICAL",
			ReflectPercent:    40,
			ReflectLtX:        -150,
			ReflectLtY:        -150,
			ReflectRbX:        150,
			ReflectRbY:        150,
			ReflectMaxDamage:  5000,
		},
	})

	ri, ok := monster.GetStatusMirror().GetReflect(tm, uniqueId, "PHYSICAL")
	if !ok {
		t.Fatalf("expected PHYSICAL reflect to be present after STATUS_APPLIED handler ran")
	}
	if ri.Percent != 40 {
		t.Fatalf("Percent: want 40, got %d", ri.Percent)
	}
	if ri.LtX != -150 || ri.LtY != -150 || ri.RbX != 150 || ri.RbY != 150 {
		t.Fatalf("reflect bounds wrong: %+v", ri)
	}
	if ri.MaxDamage != 5000 {
		t.Fatalf("MaxDamage: want 5000, got %d", ri.MaxDamage)
	}
	if _, ok := monster.GetStatusMirror().GetReflect(tm, uniqueId, "MAGIC"); ok {
		t.Fatalf("MAGIC lookup should miss when only PHYSICAL is mirrored")
	}
}
