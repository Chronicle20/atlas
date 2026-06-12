package mount

import (
	mount2 "atlas-channel/kafka/message/mount"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// tamingMobInfoBroadcast captures one invocation of the map-broadcast seam.
type tamingMobInfoBroadcast struct {
	characterId uint32
	level       uint32
	exp         uint32
	tiredness   uint32
	levelUp     bool
}

// tooTiredNotice captures one invocation of the rider-notice seam.
type tooTiredNotice struct {
	characterId uint32
	message     string
}

// withRecordingSeams swaps the package-level broadcast + notice seams for
// recording stubs so tests can assert the wire effect of the mount status
// consumer without standing up REST mocks for session resolution or
// _map.ForSessionsInMap. Returns a restore func plus pointers to the captured
// invocations.
func withRecordingSeams(t *testing.T) (restore func(), broadcasts *[]tamingMobInfoBroadcast, notices *[]tooTiredNotice) {
	t.Helper()
	var bs []tamingMobInfoBroadcast
	var ns []tooTiredNotice
	origBroadcast := tamingMobInfoBroadcaster
	origNotice := tooTiredNoticer
	tamingMobInfoBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ server.Model, characterId, level, exp, tiredness uint32, levelUp bool) {
		bs = append(bs, tamingMobInfoBroadcast{characterId, level, exp, tiredness, levelUp})
	}
	tooTiredNoticer = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ server.Model, characterId uint32, message string) {
		ns = append(ns, tooTiredNotice{characterId, message})
	}
	return func() {
		tamingMobInfoBroadcaster = origBroadcast
		tooTiredNoticer = origNotice
	}, &bs, &ns
}

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

// TestStatusEvent_SetBroadcastsTamingMobInfo asserts a SET event broadcasts
// SetTamingMobInfo to the rider's map with the converted int->uint32 stats and
// no too-tired notice.
func TestStatusEvent_SetBroadcastsTamingMobInfo(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, broadcasts, notices := withRecordingSeams(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(logrus.New(), ctx, mount2.StatusEvent[mount2.StatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: 1001,
		Type:        mount2.StatusEventTypeSet,
		Body: mount2.StatusEventBody{
			Level:     3,
			Exp:       42,
			Tiredness: 10,
			LevelUp:   true,
			TooTired:  false,
		},
	})

	if len(*broadcasts) != 1 {
		t.Fatalf("SET: want 1 SetTamingMobInfo broadcast, got %d", len(*broadcasts))
	}
	b := (*broadcasts)[0]
	if b.characterId != 1001 || b.level != 3 || b.exp != 42 || b.tiredness != 10 || !b.levelUp {
		t.Fatalf("SET broadcast wrong: %+v", b)
	}
	if len(*notices) != 0 {
		t.Fatalf("SET: want 0 notices, got %d", len(*notices))
	}
}

// TestStatusEvent_TickBroadcasts asserts a TICK without TooTired broadcasts but
// does not notice.
func TestStatusEvent_TickBroadcasts(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, broadcasts, notices := withRecordingSeams(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(logrus.New(), ctx, mount2.StatusEvent[mount2.StatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: 1002,
		Type:        mount2.StatusEventTypeTick,
		Body: mount2.StatusEventBody{Level: 5, Exp: 0, Tiredness: 50},
	})

	if len(*broadcasts) != 1 {
		t.Fatalf("TICK: want 1 broadcast, got %d", len(*broadcasts))
	}
	if len(*notices) != 0 {
		t.Fatalf("TICK (not too tired): want 0 notices, got %d", len(*notices))
	}
}

// TestStatusEvent_FeedBroadcasts asserts a FEED event broadcasts SetTamingMobInfo.
func TestStatusEvent_FeedBroadcasts(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, broadcasts, notices := withRecordingSeams(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(logrus.New(), ctx, mount2.StatusEvent[mount2.StatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: 1003,
		Type:        mount2.StatusEventTypeFeed,
		Body: mount2.StatusEventBody{Level: 2, Exp: 7, Tiredness: 0},
	})

	if len(*broadcasts) != 1 {
		t.Fatalf("FEED: want 1 broadcast, got %d", len(*broadcasts))
	}
	if len(*notices) != 0 {
		t.Fatalf("FEED: want 0 notices, got %d", len(*notices))
	}
}

// TestStatusEvent_TooTiredNoticesRider asserts a TICK with TooTired both
// broadcasts the updated info AND sends the FR-6.3 notice to the rider only.
func TestStatusEvent_TooTiredNoticesRider(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, broadcasts, notices := withRecordingSeams(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(logrus.New(), ctx, mount2.StatusEvent[mount2.StatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: 1004,
		Type:        mount2.StatusEventTypeTick,
		Body: mount2.StatusEventBody{Level: 5, Exp: 0, Tiredness: 99, TooTired: true},
	})

	if len(*broadcasts) != 1 {
		t.Fatalf("too-tired TICK: want 1 broadcast, got %d", len(*broadcasts))
	}
	if len(*notices) != 1 {
		t.Fatalf("too-tired TICK: want 1 notice, got %d", len(*notices))
	}
	n := (*notices)[0]
	if n.characterId != 1004 {
		t.Fatalf("notice characterId: want 1004, got %d", n.characterId)
	}
	if n.message != tooTiredMessage {
		t.Fatalf("notice message: want %q, got %q", tooTiredMessage, n.message)
	}
}

// TestStatusEvent_WrongWorld_DoesNothing guards the world gate.
func TestStatusEvent_WrongWorld_DoesNothing(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, broadcasts, notices := withRecordingSeams(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(logrus.New(), ctx, mount2.StatusEvent[mount2.StatusEventBody]{
		WorldId:     sc.WorldId() + 1,
		CharacterId: 1005,
		Type:        mount2.StatusEventTypeSet,
		Body:        mount2.StatusEventBody{Level: 1},
	})

	if len(*broadcasts) != 0 || len(*notices) != 0 {
		t.Fatalf("wrong world: want no effects, got %d broadcasts %d notices", len(*broadcasts), len(*notices))
	}
}

// TestStatusEvent_UnknownType_DoesNothing guards against unrelated event types
// on the topic.
func TestStatusEvent_UnknownType_DoesNothing(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, broadcasts, notices := withRecordingSeams(t)
	defer restore()

	h := handleStatusEvent(sc, nil)
	h(logrus.New(), ctx, mount2.StatusEvent[mount2.StatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: 1006,
		Type:        "UNKNOWN",
		Body:        mount2.StatusEventBody{Level: 1},
	})

	if len(*broadcasts) != 0 || len(*notices) != 0 {
		t.Fatalf("unknown type: want no effects, got %d broadcasts %d notices", len(*broadcasts), len(*notices))
	}
}
