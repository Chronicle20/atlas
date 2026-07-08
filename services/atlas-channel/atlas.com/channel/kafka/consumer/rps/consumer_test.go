package rps

import (
	rpsmsg "atlas-channel/kafka/message/rps"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	rpscb "github.com/Chronicle20/atlas/libs/atlas-packet/rps/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// rpsGameOperations mirrors the operations table the RPS_GAME writer's mode
// byte is resolved from (rps_game.yaml OPEN=8/RESULT=11/END=13 - see
// libs/atlas-packet/rps/clientbound/operation.go).
var rpsGameOperations = map[string]interface{}{
	"OPEN":   float64(8),
	"RESULT": float64(11),
	"END":    float64(13),
}

// announceCall captures one invocation of the rpsAnnouncer seam: which
// character's session it targeted, and the wire-encoded bytes produced by
// the selected writer body func.
type announceCall struct {
	characterId uint32
	bytes       []byte
}

// withRecordingAnnouncer swaps the package-level rpsAnnouncer seam for a
// recording stub that immediately invokes the passed body encoder (with a
// fixed operations table) and records the characterId + resulting bytes.
// This avoids needing a live net.Conn/session registry to assert both "which
// session was targeted" and "what body was selected" - mirrors the
// mount consumer's withRecordingSeams pattern.
func withRecordingAnnouncer(t *testing.T) (restore func(), calls *[]announceCall) {
	t.Helper()
	var recorded []announceCall
	orig := rpsAnnouncer
	l, _ := testlog.NewNullLogger()
	ctx := context.Background()
	rpsAnnouncer = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ server.Model, characterId uint32, body packet.Encode) {
		b := body(l, ctx)(map[string]interface{}{"operations": rpsGameOperations})
		recorded = append(recorded, announceCall{characterId: characterId, bytes: b})
	}
	return func() { rpsAnnouncer = orig }, &recorded
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

func decodeOpen(t *testing.T, b []byte) rpscb.Open {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var open rpscb.Open
	open.Decode(l, context.Background())(&reader, nil)
	return open
}

func decodeResult(t *testing.T, b []byte) rpscb.Result {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var result rpscb.Result
	result.Decode(l, context.Background())(&reader, nil)
	return result
}

func decodeEnd(t *testing.T, b []byte) rpscb.End {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(b)
	reader := request.NewRequestReader(&req, 0)
	var end rpscb.End
	end.Decode(l, context.Background())(&reader, nil)
	return end
}

// TestGameOpenedEvent_AnnouncesOpenWithAnte asserts a GAME_OPENED event
// selects the OPEN body func with the event's Ante and targets the
// character's session.
func TestGameOpenedEvent_AnnouncesOpenWithAnte(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleGameOpenedEvent(sc, nil)
	h(logrus.New(), ctx, rpsmsg.Event[rpsmsg.GameOpenedEventBody]{
		CharacterId: 2001,
		WorldId:     sc.WorldId(),
		ChannelId:   sc.Channel().Id(),
		Type:        rpsmsg.EventTypeGameOpened,
		Body:        rpsmsg.GameOpenedEventBody{NpcId: 9010000, Ante: 1000},
	})

	if len(*calls) != 1 {
		t.Fatalf("want 1 announce call, got %d", len(*calls))
	}
	call := (*calls)[0]
	if call.characterId != 2001 {
		t.Fatalf("want session targeted for character 2001, got %d", call.characterId)
	}
	open := decodeOpen(t, call.bytes)
	if open.Mode() != 8 {
		t.Fatalf("want resolved OPEN mode byte 8, got %d", open.Mode())
	}
	if open.Ante() != 1000 {
		t.Fatalf("want ante 1000, got %d", open.Ante())
	}
}

// TestRoundResultEvent_Win asserts a Win outcome selects RESULT with the raw
// opponent throw and a POSITIVE straightVictoryCount == Rung.
func TestRoundResultEvent_Win(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleRoundResultEvent(sc, nil)
	h(logrus.New(), ctx, rpsmsg.Event[rpsmsg.RoundResultEventBody]{
		CharacterId: 2002,
		WorldId:     sc.WorldId(),
		ChannelId:   sc.Channel().Id(),
		Type:        rpsmsg.EventTypeRoundResult,
		Body: rpsmsg.RoundResultEventBody{
			OpponentThrow: 2,
			Outcome:       rpsmsg.OutcomeWin,
			Rung:          3,
		},
	})

	if len(*calls) != 1 {
		t.Fatalf("want 1 announce call, got %d", len(*calls))
	}
	call := (*calls)[0]
	if call.characterId != 2002 {
		t.Fatalf("want session targeted for character 2002, got %d", call.characterId)
	}
	result := decodeResult(t, call.bytes)
	if result.Mode() != 11 {
		t.Fatalf("want resolved RESULT mode byte 11, got %d", result.Mode())
	}
	if result.NpcThrow() != 2 {
		t.Fatalf("want npcThrow=2 (raw OpponentThrow), got %d", result.NpcThrow())
	}
	if result.StraightVictoryCount() != 3 {
		t.Fatalf("win: want straightVictoryCount=+3, got %d", result.StraightVictoryCount())
	}
}

// TestRoundResultEvent_Tie asserts a Tie outcome also yields a
// non-negative straightVictoryCount == Rung (unchanged streak).
func TestRoundResultEvent_Tie(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleRoundResultEvent(sc, nil)
	h(logrus.New(), ctx, rpsmsg.Event[rpsmsg.RoundResultEventBody]{
		CharacterId: 2003,
		WorldId:     sc.WorldId(),
		ChannelId:   sc.Channel().Id(),
		Type:        rpsmsg.EventTypeRoundResult,
		Body: rpsmsg.RoundResultEventBody{
			OpponentThrow: 0,
			Outcome:       rpsmsg.OutcomeTie,
			Rung:          2,
		},
	})

	result := decodeResult(t, (*calls)[0].bytes)
	if result.StraightVictoryCount() != 2 {
		t.Fatalf("tie: want straightVictoryCount=+2 (unchanged Rung), got %d", result.StraightVictoryCount())
	}
	if result.StraightVictoryCount() < 0 {
		t.Fatalf("tie: straightVictoryCount must not be negative, got %d", result.StraightVictoryCount())
	}
}

// TestRoundResultEvent_Lose asserts a Lose outcome yields a NEGATIVE
// straightVictoryCount - the client keys "lose" solely on the sign
// (IDA-verified; magnitude is display-only, we use -1).
func TestRoundResultEvent_Lose(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleRoundResultEvent(sc, nil)
	h(logrus.New(), ctx, rpsmsg.Event[rpsmsg.RoundResultEventBody]{
		CharacterId: 2004,
		WorldId:     sc.WorldId(),
		ChannelId:   sc.Channel().Id(),
		Type:        rpsmsg.EventTypeRoundResult,
		Body: rpsmsg.RoundResultEventBody{
			OpponentThrow: 1,
			Outcome:       rpsmsg.OutcomeLose,
			Rung:          5,
		},
	})

	if len(*calls) != 1 {
		t.Fatalf("want 1 announce call, got %d", len(*calls))
	}
	result := decodeResult(t, (*calls)[0].bytes)
	if result.NpcThrow() != 1 {
		t.Fatalf("want npcThrow=1 (raw OpponentThrow), got %d", result.NpcThrow())
	}
	if result.StraightVictoryCount() >= 0 {
		t.Fatalf("lose: want NEGATIVE straightVictoryCount, got %d", result.StraightVictoryCount())
	}
}

// TestGameEndedEvent_AnnouncesEndBodyless asserts a GAME_ENDED event selects
// the bodyless END writer and targets the character's session.
func TestGameEndedEvent_AnnouncesEndBodyless(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleGameEndedEvent(sc, nil)
	h(logrus.New(), ctx, rpsmsg.Event[rpsmsg.GameEndedEventBody]{
		CharacterId: 2005,
		WorldId:     sc.WorldId(),
		ChannelId:   sc.Channel().Id(),
		Type:        rpsmsg.EventTypeGameEnded,
		Body:        rpsmsg.GameEndedEventBody{Reason: rpsmsg.ReasonCollected},
	})

	if len(*calls) != 1 {
		t.Fatalf("want 1 announce call, got %d", len(*calls))
	}
	call := (*calls)[0]
	if call.characterId != 2005 {
		t.Fatalf("want session targeted for character 2005, got %d", call.characterId)
	}
	end := decodeEnd(t, call.bytes)
	if end.Mode() != 13 {
		t.Fatalf("want resolved END mode byte 13, got %d", end.Mode())
	}
}

// TestRoundResultEvent_WrongChannel_DoesNothing guards the tenant/world/
// channel gate.
func TestRoundResultEvent_WrongChannel_DoesNothing(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleRoundResultEvent(sc, nil)
	h(logrus.New(), ctx, rpsmsg.Event[rpsmsg.RoundResultEventBody]{
		CharacterId: 2006,
		WorldId:     sc.WorldId(),
		ChannelId:   sc.Channel().Id() + 1,
		Type:        rpsmsg.EventTypeRoundResult,
		Body:        rpsmsg.RoundResultEventBody{OpponentThrow: 0, Outcome: rpsmsg.OutcomeWin, Rung: 1},
	})

	if len(*calls) != 0 {
		t.Fatalf("wrong channel: want no effects, got %d", len(*calls))
	}
}

// TestRoundResultEvent_UnknownType_DoesNothing guards against unrelated
// event types on the shared topic.
func TestRoundResultEvent_UnknownType_DoesNothing(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingAnnouncer(t)
	defer restore()

	h := handleRoundResultEvent(sc, nil)
	h(logrus.New(), ctx, rpsmsg.Event[rpsmsg.RoundResultEventBody]{
		CharacterId: 2007,
		WorldId:     sc.WorldId(),
		ChannelId:   sc.Channel().Id(),
		Type:        "UNKNOWN",
		Body:        rpsmsg.RoundResultEventBody{OpponentThrow: 0, Outcome: rpsmsg.OutcomeWin, Rung: 1},
	})

	if len(*calls) != 0 {
		t.Fatalf("unknown type: want no effects, got %d", len(*calls))
	}
}
