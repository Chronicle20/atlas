package buff

import (
	"atlas-channel/battleship"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"testing"
	"time"

	battleshipmock "atlas-channel/battleship/mock"
	buff2 "atlas-channel/kafka/message/buff"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestIsBattleshipRide(t *testing.T) {
	riding := []buff2.StatChange{{Type: "MONSTER_RIDING", Amount: 1932000}}
	noRiding := []buff2.StatChange{{Type: "WEAPON_DEFENSE", Amount: 10}}
	tests := []struct {
		name     string
		sourceId int32
		changes  []buff2.StatChange
		expected bool
	}{
		{"battleship riding buff", 5221006, riding, true},
		{"battleship without riding change", 5221006, noRiding, false},
		{"other mount riding buff", 1019, riding, false},
		{"cannon is not the mount", 5221007, riding, false},
		{"empty changes", 5221006, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isBattleshipRide(tc.sourceId, tc.changes); got != tc.expected {
				t.Errorf("isBattleshipRide(%d, %v) = %v, want %v", tc.sourceId, tc.changes, got, tc.expected)
			}
		})
	}
}

// --- lifecycle-hook wiring tests (battleship ride begin/end) ---
//
// These cover the three real call sites the pure isBattleshipRide predicate
// above does not: the mirror Put in handleStatusEventApplied, and the Clear
// in handleStatusEventExpired. (The third call site, session.Destroy, is
// covered in session/battleship_hook_test.go — a different package.)

// newTestTenant returns a fresh, isolated tenant per test so the process-wide
// battleship.RideMirror singleton and session registry can never bleed
// state between test cases.
func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// testLogger returns a discard-output logger so expected error-path logging
// (e.g. the stubbed writer producer below) does not clutter test output.
func testLogger() *logrus.Logger {
	l, _ := logtest.NewNullLogger()
	return l
}

// newTestServer registers a server.Model at world 0 / channel 0 — matching
// the default world/channel of a session.NewSession-created test session
// (session.Model does not set a field until SetField is called), so
// IfPresentByCharacterId can find it.
func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 0)
	return server.NewProcessor(testLogger(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// registerTestSession registers a session for characterId in the local,
// in-memory session registry (matching sc's world/channel), so
// IfPresentByCharacterId succeeds the way it would on the channel pod that
// actually owns the character's socket. Returns a cleanup func.
func registerTestSession(t *testing.T, tm tenant.Model, characterId uint32) func() {
	t.Helper()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, tm, 0, nil)
	session.AddSessionToRegistry(tm.Id(), s)
	ctx := tenant.WithContext(context.Background(), tm)
	if r := session.NewProcessor(testLogger(), ctx).SetCharacterId(sessionId, characterId); r.CharacterId() != characterId {
		t.Fatalf("SetCharacterId: got %d, want %d", r.CharacterId(), characterId)
	}
	return func() { session.ClearRegistryForTenant(tm.Id()) }
}

// noOpWriterProducer stubs writer.Producer: it always errors, so
// session.Announce logs a warning and returns without touching the (nil)
// socket connection on our test sessions. The announce path is not what
// these tests are pinning; the battleship hooks that run BEFORE the
// announce are.
func noOpWriterProducer(_ string) (writer.BodyFunc, error) {
	return nil, errors.New("no writer configured (test stub)")
}

// battleshipSpy records battleship.Processor invocations via a
// battleshipmock.ProcessorMock, so a test can assert exactly which methods a
// hook called — most importantly, that Drain (the only path that can reach
// breakShip's cooldown emit) was never touched by a lifecycle hook that is
// only supposed to Clear.
type battleshipSpy struct {
	clearCalls      []uint32
	drainCalls      int
	initShipHPCalls int
	isRidingCalls   int
}

func newBattleshipSpy() (*battleshipmock.ProcessorMock, *battleshipSpy) {
	spy := &battleshipSpy{}
	m := &battleshipmock.ProcessorMock{
		ClearFunc: func(characterId uint32) { spy.clearCalls = append(spy.clearCalls, characterId) },
		DrainFunc: func(_ field.Model, _ uint32, _ int32) battleship.DrainResult {
			spy.drainCalls++
			return battleship.DrainResult{}
		},
		InitShipHPFunc: func(_ uint32, _ byte, _ byte, _ time.Duration) error {
			spy.initShipHPCalls++
			return nil
		},
		IsRidingFunc: func(_ uint32) (byte, bool) {
			spy.isRidingCalls++
			return 0, false
		},
	}
	return m, spy
}

func (s *battleshipSpy) assertOnlyClear(t *testing.T) {
	t.Helper()
	if s.drainCalls != 0 || s.initShipHPCalls != 0 || s.isRidingCalls != 0 {
		t.Errorf("hook touched the Processor beyond Clear (cooldown-reachable surface): drain=%d initShipHP=%d isRiding=%d",
			s.drainCalls, s.initShipHPCalls, s.isRidingCalls)
	}
}

// TestHandleStatusEventApplied_BattleshipRide_PutsRideState pins Task 7's
// ride-begin edge: an APPLIED event carrying the battleship's MONSTER_RIDING
// change must Put the event's Level and the TTL-seam's derived duration into
// the mirror, and must never touch the battleship.Processor / cooldown
// surface (Put is a pure mirror write).
func TestHandleStatusEventApplied_BattleshipRide_PutsRideState(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	characterId := uint32(5001)

	cleanupSession := registerTestSession(t, tm, characterId)
	defer cleanupSession()
	defer battleship.GetRideMirror().Remove(tm, characterId)

	origTTLFunc := battleshipStateTTLFunc
	wantTTL := 42 * time.Minute
	battleshipStateTTLFunc = func(_ logrus.FieldLogger, _ context.Context, _ byte) time.Duration { return wantTTL }
	defer func() { battleshipStateTTLFunc = origTTLFunc }()

	origNewProcessor := newBattleshipProcessor
	mock, spy := newBattleshipSpy()
	newBattleshipProcessor = func(_ logrus.FieldLogger, _ context.Context) battleship.Processor { return mock }
	defer func() { newBattleshipProcessor = origNewProcessor }()

	h := handleStatusEventApplied(sc, noOpWriterProducer)
	h(testLogger(), ctx, buff2.StatusEvent[buff2.AppliedStatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: characterId,
		Type:        buff2.EventStatusTypeBuffApplied,
		Body: buff2.AppliedStatusEventBody{
			SourceId: 5221006,
			Level:    7,
			Changes:  []buff2.StatChange{{Type: "MONSTER_RIDING", Amount: 1932000}},
		},
	})

	rs, ok := battleship.GetRideMirror().Get(tm, characterId)
	if !ok {
		t.Fatal("expected a ride state to be Put into the mirror, found none")
	}
	if rs.SkillLevel != 7 {
		t.Errorf("SkillLevel = %d, want 7", rs.SkillLevel)
	}
	if rs.StateTTL != wantTTL {
		t.Errorf("StateTTL = %v, want %v", rs.StateTTL, wantTTL)
	}

	// Structural pin: the ride-begin hook only ever calls
	// battleship.GetRideMirror().Put directly — it must never construct a
	// battleship.Processor at all, let alone reach Drain/cooldown.
	if len(spy.clearCalls) != 0 {
		t.Errorf("Clear calls = %v, want none from the ride-begin hook", spy.clearCalls)
	}
	spy.assertOnlyClear(t)
}

// TestHandleStatusEventApplied_NonBattleshipBuff_NoPut pins the negative
// case: an APPLIED event for an unrelated buff must not touch the mirror.
func TestHandleStatusEventApplied_NonBattleshipBuff_NoPut(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	characterId := uint32(5002)

	cleanupSession := registerTestSession(t, tm, characterId)
	defer cleanupSession()
	defer battleship.GetRideMirror().Remove(tm, characterId)

	h := handleStatusEventApplied(sc, noOpWriterProducer)
	h(testLogger(), ctx, buff2.StatusEvent[buff2.AppliedStatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: characterId,
		Type:        buff2.EventStatusTypeBuffApplied,
		Body: buff2.AppliedStatusEventBody{
			SourceId: 1002,
			Level:    1,
			Changes:  []buff2.StatChange{{Type: "WEAPON_DEFENSE", Amount: 10}},
		},
	})

	if _, ok := battleship.GetRideMirror().Get(tm, characterId); ok {
		t.Error("expected no ride state for a non-battleship buff, but the mirror has one")
	}
}

// TestHandleStatusEventExpired_BattleshipRide_ClearsState_NoCooldown pins
// Task 7's ride-end edge: an EXPIRED event carrying the battleship's
// MONSTER_RIDING change must call Clear exactly once for the character, and
// must NEVER call Drain — the only battleship.Processor path that can reach
// breakShip's cooldown emit. Covers all three EXPIRED sources (manual
// dismount, server cancel on break, natural expiry) since they are
// wire-identical from the consumer's point of view.
func TestHandleStatusEventExpired_BattleshipRide_ClearsState_NoCooldown(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	characterId := uint32(5003)

	cleanupSession := registerTestSession(t, tm, characterId)
	defer cleanupSession()

	origNewProcessor := newBattleshipProcessor
	mock, spy := newBattleshipSpy()
	newBattleshipProcessor = func(_ logrus.FieldLogger, _ context.Context) battleship.Processor { return mock }
	defer func() { newBattleshipProcessor = origNewProcessor }()

	h := handleStatusEventExpired(sc, noOpWriterProducer)
	h(testLogger(), ctx, buff2.StatusEvent[buff2.ExpiredStatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: characterId,
		Type:        buff2.EventStatusTypeBuffExpired,
		Body: buff2.ExpiredStatusEventBody{
			SourceId: 5221006,
			Level:    7,
			Changes:  []buff2.StatChange{{Type: "MONSTER_RIDING", Amount: 1932000}},
		},
	})

	if len(spy.clearCalls) != 1 || spy.clearCalls[0] != characterId {
		t.Fatalf("Clear calls = %v, want exactly one Clear(%d)", spy.clearCalls, characterId)
	}
	spy.assertOnlyClear(t)
}

// TestHandleStatusEventExpired_NonBattleshipBuff_NoClear pins the negative
// case: an EXPIRED event for an unrelated buff must not call Clear.
func TestHandleStatusEventExpired_NonBattleshipBuff_NoClear(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	characterId := uint32(5004)

	cleanupSession := registerTestSession(t, tm, characterId)
	defer cleanupSession()

	origNewProcessor := newBattleshipProcessor
	mock, spy := newBattleshipSpy()
	newBattleshipProcessor = func(_ logrus.FieldLogger, _ context.Context) battleship.Processor { return mock }
	defer func() { newBattleshipProcessor = origNewProcessor }()

	h := handleStatusEventExpired(sc, noOpWriterProducer)
	h(testLogger(), ctx, buff2.StatusEvent[buff2.ExpiredStatusEventBody]{
		WorldId:     sc.WorldId(),
		CharacterId: characterId,
		Type:        buff2.EventStatusTypeBuffExpired,
		Body: buff2.ExpiredStatusEventBody{
			SourceId: 1002,
			Level:    1,
			Changes:  []buff2.StatChange{{Type: "WEAPON_DEFENSE", Amount: 10}},
		},
	})

	if len(spy.clearCalls) != 0 {
		t.Errorf("Clear calls = %v, want none for a non-battleship buff", spy.clearCalls)
	}
}
