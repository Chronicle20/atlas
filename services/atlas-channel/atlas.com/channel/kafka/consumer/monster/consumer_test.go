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
