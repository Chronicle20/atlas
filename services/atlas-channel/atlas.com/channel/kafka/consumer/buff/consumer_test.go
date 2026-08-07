package buff

import (
	"atlas-channel/battleship"
	"atlas-channel/character/buff"
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
	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
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
// above does not: the battleship.Processor.StartRide call in
// handleStatusEventApplied, and the Clear call in handleStatusEventExpired.
// (The third call site, session.Destroy, is covered in
// session/battleship_hook_test.go — a different package.)

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

// startRideCall pins one StartRide(characterId, state) invocation observed
// by battleshipSpy.
type startRideCall struct {
	characterId uint32
	state       battleship.RideState
}

// battleshipSpy records battleship.Processor invocations via a
// battleshipmock.ProcessorMock, so a test can assert exactly which methods a
// hook called — most importantly, that Drain (the only path that can reach
// breakShip's cooldown emit) was never touched by a lifecycle hook that is
// only supposed to Clear or StartRide.
type battleshipSpy struct {
	clearCalls      []uint32
	drainCalls      int
	initShipHPCalls int
	isRidingCalls   int
	startRideCalls  []startRideCall
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
		StartRideFunc: func(characterId uint32, s battleship.RideState) {
			spy.startRideCalls = append(spy.startRideCalls, startRideCall{characterId: characterId, state: s})
		},
	}
	return m, spy
}

func (s *battleshipSpy) assertOnlyClear(t *testing.T) {
	t.Helper()
	if s.drainCalls != 0 || s.initShipHPCalls != 0 || s.isRidingCalls != 0 || len(s.startRideCalls) != 0 {
		t.Errorf("hook touched the Processor beyond Clear (cooldown-reachable surface): drain=%d initShipHP=%d isRiding=%d startRide=%d",
			s.drainCalls, s.initShipHPCalls, s.isRidingCalls, len(s.startRideCalls))
	}
}

// assertOnlyStartRide is assertOnlyClear's counterpart for the ride-begin
// hook: it must touch StartRide and nothing else on the cooldown-reachable
// surface (Clear/Drain/InitShipHP/IsRiding).
func (s *battleshipSpy) assertOnlyStartRide(t *testing.T) {
	t.Helper()
	if s.drainCalls != 0 || s.initShipHPCalls != 0 || s.isRidingCalls != 0 || len(s.clearCalls) != 0 {
		t.Errorf("hook touched the Processor beyond StartRide (cooldown-reachable surface): drain=%d initShipHP=%d isRiding=%d clear=%v",
			s.drainCalls, s.initShipHPCalls, s.isRidingCalls, s.clearCalls)
	}
}

// TestHandleStatusEventApplied_BattleshipRide_CallsStartRide pins Task 7's
// ride-begin edge (as fixed by the task-153 final-review Finding 1 pass): an
// APPLIED event carrying the battleship's MONSTER_RIDING change must call
// battleship.Processor.StartRide exactly once with the event's Level and the
// TTL-seam's derived duration, routed through the same newBattleshipProcessor
// seam as the EXPIRED hook's Clear — and must never touch Clear/Drain/
// InitShipHP/IsRiding.
func TestHandleStatusEventApplied_BattleshipRide_CallsStartRide(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	characterId := uint32(5001)

	cleanupSession := registerTestSession(t, tm, characterId)
	defer cleanupSession()

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

	if len(spy.startRideCalls) != 1 {
		t.Fatalf("StartRide calls = %d, want exactly 1", len(spy.startRideCalls))
	}
	call := spy.startRideCalls[0]
	if call.characterId != characterId {
		t.Errorf("StartRide characterId = %d, want %d", call.characterId, characterId)
	}
	if call.state.SkillLevel != 7 {
		t.Errorf("StartRide SkillLevel = %d, want 7", call.state.SkillLevel)
	}
	if call.state.StateTTL != wantTTL {
		t.Errorf("StartRide StateTTL = %v, want %v", call.state.StateTTL, wantTTL)
	}

	// Structural pin: the ride-begin hook must call StartRide and nothing
	// else on the Processor's cooldown-reachable surface.
	spy.assertOnlyStartRide(t)
}

// TestHandleStatusEventApplied_NonBattleshipBuff_NoStartRide pins the
// negative case: an APPLIED event for an unrelated buff must not touch the
// mirror (isBattleshipRide gates the StartRide call before any Processor is
// constructed, so this test does not need to install the newBattleshipProcessor
// seam).
func TestHandleStatusEventApplied_NonBattleshipBuff_NoStartRide(t *testing.T) {
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

// --- beacon merge / foreign-suppression pure-helper tests (task-167 F2/FR-4.5) ---

func TestBeaconChange(t *testing.T) {
	_, ok := beaconChange([]buff2.StatChange{{Type: "SPEED", Amount: 20}})
	if ok {
		t.Fatal("no beacon change expected")
	}
	c, ok := beaconChange([]buff2.StatChange{
		{Type: "SPEED", Amount: 20},
		{Type: string(charconst.TemporaryStatTypeHomingBeacon), Amount: 1000001},
	})
	if !ok || c.Amount != 1000001 {
		t.Fatalf("beacon change: got %+v ok=%v", c, ok)
	}
}

func TestIsBeaconOnly(t *testing.T) {
	if isBeaconOnly(nil) {
		t.Fatal("empty changes are not beacon-only")
	}
	if isBeaconOnly([]buff2.StatChange{{Type: "SPEED", Amount: 20}}) {
		t.Fatal("SPEED is not beacon-only")
	}
	if !isBeaconOnly([]buff2.StatChange{{Type: string(charconst.TemporaryStatTypeHomingBeacon), Amount: 1000001}}) {
		t.Fatal("single HOMING_BEACON change is beacon-only")
	}
	if isBeaconOnly([]buff2.StatChange{
		{Type: string(charconst.TemporaryStatTypeHomingBeacon), Amount: 1000001},
		{Type: "SPEED", Amount: 20},
	}) {
		t.Fatal("mixed changes are not beacon-only")
	}
}

func TestMergeBeacon(t *testing.T) {
	bs := []buff.Model{}
	out := mergeBeacon(bs, buff.NewBeaconEntry(5211006, 1, 1000001))
	if len(out) != 1 {
		t.Fatalf("merge: got %d buffs want 1", len(out))
	}
	b := out[0]
	if b.SourceId() != 5211006 || !b.NoExpiry() {
		t.Fatalf("merged beacon buff wrong: sourceId=%d noExpiry=%v", b.SourceId(), b.NoExpiry())
	}
	if len(b.Changes()) != 1 || b.Changes()[0].Type() != string(charconst.TemporaryStatTypeHomingBeacon) || b.Changes()[0].Amount() != 1000001 {
		t.Fatalf("merged beacon statup wrong: %+v", b.Changes())
	}
}
