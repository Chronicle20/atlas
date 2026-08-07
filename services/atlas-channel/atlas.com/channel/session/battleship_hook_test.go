package session

import (
	"atlas-channel/battleship"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	battleshipmock "atlas-channel/battleship/mock"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// This file is a white-box (package session) test, deliberately separate
// from processor_test.go (package session_test): it needs direct access to
// the unexported newBattleshipProcessor seam to assert exactly which
// battleship.Processor methods session.Destroy's ride-cleanup hook invokes.
// It calls clearBattleshipOnDestroy directly rather than the full Destroy —
// Destroy also emits a logout command and a DESTROYED status event over a
// real Kafka producer (producer.ProviderImpl dials os.Getenv
// ("BOOTSTRAP_SERVERS") with up to 10 retries), which would make this test
// slow/flaky for no benefit: clearBattleshipOnDestroy is exactly the hook
// Task 7 added, extracted so it's testable in isolation.

// battleshipSpy records battleship.Processor invocations via a
// battleshipmock.ProcessorMock, so a test can assert exactly which methods a
// hook called — most importantly, that Drain (the only path that can reach
// breakShip's cooldown emit) was never touched by a hook that is only
// supposed to Clear.
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

// TestClearBattleshipOnDestroy_NonZeroCharacter_ClearsState pins Task 7's
// session-destroy edge: a destroyed session with a real character must call
// Clear exactly once for that character, and must NEVER call Drain — the
// only battleship.Processor path that can reach breakShip's cooldown emit.
func TestClearBattleshipOnDestroy_NonZeroCharacter_ClearsState(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	orig := newBattleshipProcessor
	mock, spy := newBattleshipSpy()
	newBattleshipProcessor = func(_ logrus.FieldLogger, _ context.Context) battleship.Processor { return mock }
	defer func() { newBattleshipProcessor = orig }()

	clearBattleshipOnDestroy(logger, ctx, 4242)

	if len(spy.clearCalls) != 1 || spy.clearCalls[0] != 4242 {
		t.Fatalf("Clear calls = %v, want exactly one Clear(4242)", spy.clearCalls)
	}
	spy.assertOnlyClear(t)
}

// TestClearBattleshipOnDestroy_ZeroCharacter_NoOp pins the negative case: a
// session that never reached character selection (CharacterId == 0) must not
// touch the battleship.Processor at all.
func TestClearBattleshipOnDestroy_ZeroCharacter_NoOp(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	orig := newBattleshipProcessor
	mock, spy := newBattleshipSpy()
	newBattleshipProcessor = func(_ logrus.FieldLogger, _ context.Context) battleship.Processor { return mock }
	defer func() { newBattleshipProcessor = orig }()

	clearBattleshipOnDestroy(logger, ctx, 0)

	if len(spy.clearCalls) != 0 {
		t.Errorf("Clear calls = %v, want none when CharacterId is 0", spy.clearCalls)
	}
	spy.assertOnlyClear(t)
}
