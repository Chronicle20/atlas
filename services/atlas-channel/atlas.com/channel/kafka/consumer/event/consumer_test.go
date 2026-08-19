package event

import (
	event2 "atlas-channel/kafka/message/event"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// withRecordingBroadcasters swaps the package-level broadcast seams for
// recording stubs that capture invocations. Returns a restore func and
// pointers to the captured invocation counts / last-seen arguments. Tests
// use this to assert the ContiMove / background-music wire effect of the
// event-visual consumer without standing up a REST mock for
// _map.ForSessionsInMap.
//
// contiMoveBroadcaster now carries a writer.ContiMoveKey (SHOW/HIDE) rather
// than raw state/subState bytes -- those are client wire bytes resolved
// tenant-side from the ContiMove writer options table (DOM-25), not carried
// on the event. The recording stub captures the key instead.
func withRecordingBroadcasters(t *testing.T) (restore func(), contiMoveCalls *int, lastContiMoveKey *writer.ContiMoveKey, bgmCalls *int, lastBgm *string) {
	t.Helper()
	contiMoveN, bgmN := 0, 0
	var capturedKey writer.ContiMoveKey
	var capturedBgm string

	origContiMove := contiMoveBroadcaster
	origBgm := backgroundMusicBroadcaster

	contiMoveBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ field.Model, key writer.ContiMoveKey) {
		contiMoveN++
		capturedKey = key
	}
	backgroundMusicBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ field.Model, bgm string) {
		bgmN++
		capturedBgm = bgm
	}

	return func() {
		contiMoveBroadcaster = origContiMove
		backgroundMusicBroadcaster = origBgm
	}, &contiMoveN, &capturedKey, &bgmN, &capturedBgm
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model, worldId world.Id, channelId channel.Id) server.Model {
	t.Helper()
	ch := channel.NewModel(worldId, channelId)
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// TestShowVisualBroadcastsContiMoveToTheMap asserts a SHOW event for
// CONTI_MOVE, addressed to this channel, is translated into a ContiMove
// broadcast (and a background-music broadcast, since the seed carries one).
func TestShowVisualBroadcastsContiMoveToTheMap(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm, 1, 4)

	restore, contiMoveCalls, lastKey, bgmCalls, lastBgm := withRecordingBroadcasters(t)
	defer restore()

	handleVisualShow(sc, nil)(logrus.New(), ctx, event2.VisualEvent[event2.ShowVisualBody]{
		OccurrenceId: uuid.New(),
		WorldId:      sc.WorldId(),
		ChannelId:    sc.ChannelId(),
		MapId:        _map.Id(200090010),
		Type:         event2.VisualTypeShow,
		Body: event2.ShowVisualBody{
			Visual: event2.VisualContiMove,
			Bgm:    "Bgm04/ArabPirate",
		},
	})

	if *contiMoveCalls != 1 {
		t.Fatalf("ContiMove broadcast %d times, want 1", *contiMoveCalls)
	}
	if *lastKey != writer.ContiMoveShow {
		t.Fatalf("ContiMove key: want %q, got %q", writer.ContiMoveShow, *lastKey)
	}
	if *bgmCalls != 1 {
		t.Fatalf("FieldEffect (bgm) broadcast %d times, want 1", *bgmCalls)
	}
	if *lastBgm != "Bgm04/ArabPirate" {
		t.Fatalf("bgm: want Bgm04/ArabPirate, got %s", *lastBgm)
	}
}

// TestShowVisualIgnoresOtherChannels guards the sc.Is channel/world guard --
// an event for another channel must produce no broadcast at all.
func TestShowVisualIgnoresOtherChannels(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm, 1, 4)

	restore, contiMoveCalls, _, bgmCalls, _ := withRecordingBroadcasters(t)
	defer restore()

	handleVisualShow(sc, nil)(logrus.New(), ctx, event2.VisualEvent[event2.ShowVisualBody]{
		OccurrenceId: uuid.New(),
		WorldId:      sc.WorldId(),
		ChannelId:    5, // not this channel's 4
		MapId:        _map.Id(200090010),
		Type:         event2.VisualTypeShow,
		Body: event2.ShowVisualBody{
			Visual: event2.VisualContiMove,
		},
	})

	if *contiMoveCalls != 0 || *bgmCalls != 0 {
		t.Fatalf("other channel: want 0 broadcasts, got contiMove=%d bgm=%d", *contiMoveCalls, *bgmCalls)
	}
}

// TestShowVisualWithoutBgmSendsOnlyTheVisual asserts a SHOW with no bgm
// configured (the field is omitempty on the wire) sends the visual and
// nothing else.
func TestShowVisualWithoutBgmSendsOnlyTheVisual(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm, 1, 4)

	restore, contiMoveCalls, _, bgmCalls, _ := withRecordingBroadcasters(t)
	defer restore()

	handleVisualShow(sc, nil)(logrus.New(), ctx, event2.VisualEvent[event2.ShowVisualBody]{
		OccurrenceId: uuid.New(),
		WorldId:      sc.WorldId(),
		ChannelId:    sc.ChannelId(),
		MapId:        _map.Id(200090010),
		Type:         event2.VisualTypeShow,
		Body: event2.ShowVisualBody{
			Visual: event2.VisualContiMove,
			Bgm:    "",
		},
	})

	if *contiMoveCalls != 1 {
		t.Fatalf("ContiMove broadcast %d times, want 1", *contiMoveCalls)
	}
	if *bgmCalls != 0 {
		t.Fatalf("bgm broadcast %d times, want 0 when Bgm is empty", *bgmCalls)
	}
}

// TestHideVisualBroadcastsContiMove asserts a HIDE event broadcasts the
// ContiMove wire effect and never touches the music -- HideVisualBody
// carries no BGM field, and the handler must not invent a stop.
func TestHideVisualBroadcastsContiMove(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm, 1, 4)

	restore, contiMoveCalls, lastKey, bgmCalls, _ := withRecordingBroadcasters(t)
	defer restore()

	handleVisualHide(sc, nil)(logrus.New(), ctx, event2.VisualEvent[event2.HideVisualBody]{
		OccurrenceId: uuid.New(),
		WorldId:      sc.WorldId(),
		ChannelId:    sc.ChannelId(),
		MapId:        _map.Id(200090010),
		Type:         event2.VisualTypeHide,
		Body: event2.HideVisualBody{
			Visual: event2.VisualContiMove,
		},
	})

	if *contiMoveCalls != 1 {
		t.Fatalf("ContiMove broadcast %d times, want 1", *contiMoveCalls)
	}
	if *lastKey != writer.ContiMoveHide {
		t.Fatalf("ContiMove key: want %q, got %q", writer.ContiMoveHide, *lastKey)
	}
	if *bgmCalls != 0 {
		t.Fatalf("HIDE must never touch the music, got %d bgm broadcasts", *bgmCalls)
	}
}

// TestShowVisualWrongTypeDoesNotBroadcast guards against the SHOW handler
// firing for an unrelated event type delivered on the same topic.
func TestShowVisualWrongTypeDoesNotBroadcast(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm, 1, 4)

	restore, contiMoveCalls, _, bgmCalls, _ := withRecordingBroadcasters(t)
	defer restore()

	handleVisualShow(sc, nil)(logrus.New(), ctx, event2.VisualEvent[event2.ShowVisualBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		MapId:     _map.Id(200090010),
		Type:      event2.VisualTypeHide, // wrong type for the SHOW handler
		Body:      event2.ShowVisualBody{Visual: event2.VisualContiMove},
	})

	if *contiMoveCalls != 0 || *bgmCalls != 0 {
		t.Fatalf("wrong-type event: want 0 broadcasts, got contiMove=%d bgm=%d", *contiMoveCalls, *bgmCalls)
	}
}

// TestShowVisualUnknownVisualIsIgnored asserts an unrecognised visual name
// is ignored rather than broadcast under the ContiMove writer.
func TestShowVisualUnknownVisualIsIgnored(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm, 1, 4)

	restore, contiMoveCalls, _, bgmCalls, _ := withRecordingBroadcasters(t)
	defer restore()

	handleVisualShow(sc, nil)(logrus.New(), ctx, event2.VisualEvent[event2.ShowVisualBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		MapId:     _map.Id(200090010),
		Type:      event2.VisualTypeShow,
		Body:      event2.ShowVisualBody{Visual: "SOME_FUTURE_VISUAL"},
	})

	if *contiMoveCalls != 0 || *bgmCalls != 0 {
		t.Fatalf("unknown visual: want 0 broadcasts, got contiMove=%d bgm=%d", *contiMoveCalls, *bgmCalls)
	}
}
