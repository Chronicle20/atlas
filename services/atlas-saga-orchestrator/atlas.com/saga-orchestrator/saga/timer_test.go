package saga

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestTimerRegistry_ScheduleAndFire(t *testing.T) {
	ResetCache()
	SagaTimers().Cancel(uuid.UUID{}) // no-op, just exercises the empty path
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	_ = GetCache().Put(ctx, s)

	SagaTimers().Schedule(logger, te, s.TransactionId(), 50*time.Millisecond)
	assert.True(t, SagaTimers().Has(s.TransactionId()))

	// Wait for the timer to fire and the registry self-cleanup to run.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !SagaTimers().Has(s.TransactionId()) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	assert.False(t, SagaTimers().Has(s.TransactionId()), "timer should have self-cleaned after firing")

	// After the timer fires, handleSagaTimeout walks the full flow:
	//   Pending → Compensating → (dispatch rollbacks) → Failed → evict.
	// The saga no longer exists in the cache, so GetLifecycle returns
	// (zero, false). This verifies the bug fix: prior to the fix the timer
	// stopped at Compensating and left the saga in cache forever.
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, ok := GetCache().GetLifecycle(ctx, s.TransactionId()); !ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	_, ok := GetCache().GetLifecycle(ctx, s.TransactionId())
	assert.False(t, ok, "saga should be evicted after timer finalization")
}

func TestTimerRegistry_CancelPreventsFire(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	_ = GetCache().Put(ctx, s)

	SagaTimers().Schedule(logger, te, s.TransactionId(), 200*time.Millisecond)
	SagaTimers().Cancel(s.TransactionId())

	time.Sleep(300 * time.Millisecond)

	// Should still be Pending — the timer was cancelled before it fired.
	state, _ := GetCache().GetLifecycle(ctx, s.TransactionId())
	assert.Equal(t, SagaLifecyclePending, state)
}

func TestTimerRegistry_ScheduleReplacesExisting(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	_ = GetCache().Put(ctx, s)

	// First schedule 10s — would not fire in test window.
	SagaTimers().Schedule(logger, te, s.TransactionId(), 10*time.Second)
	// Replace with 30ms.
	SagaTimers().Schedule(logger, te, s.TransactionId(), 30*time.Millisecond)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !SagaTimers().Has(s.TransactionId()) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	assert.False(t, SagaTimers().Has(s.TransactionId()))
}

func TestTimerRegistry_ZeroDurationNoOp(t *testing.T) {
	logger, _ := test.NewNullLogger()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	txId := uuid.New()

	SagaTimers().Schedule(logger, te, txId, 0)
	assert.False(t, SagaTimers().Has(txId))
}

// TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout guards the class of
// bug that TradeStaging fell into, rather than that one instance.
//
// Timeout routing and step-failure routing are two separate enumerations of the
// saga types. A type registered in one and not the other still behaves
// correctly on step failure — which is what gets exercised in development — and
// destroys value only on the 30s backstop, which does not. TradeStaging was
// exactly that: its release_from_character completed, its accept_to_trade
// stalled, and nothing rolled back, so the compartment lost an item that escrow
// never received.
//
// So the assertion is not "TradeStaging is handled" but "every type that has a
// bespoke compensator is handled", which fails for the NEXT type added too.
func TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout(t *testing.T) {
	logger, _ := test.NewNullLogger()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	// Deliberately no ResetCache/Put. The saga carries no completed steps, so
	// every dispatch arm's reverse walk is a no-op loop and the routing decision
	// is all that is exercised. Touching the shared cache here would race the
	// timers other tests in this package leave in flight — which -race catches
	// only in a full-package run, not when this test is run alone.
	for _, st := range reverseWalkSagaTypes {
		t.Run(string(st), func(t *testing.T) {
			s, _ := NewBuilder().SetSagaType(st).SetInitiatedBy("test").Build()
			if !dispatchTimeoutRollbacks(logger, ctx, s) {
				t.Fatalf("a timed-out %s saga dispatches no reverse walk; its completed steps stand and whatever they moved is destroyed", st)
			}
		})
	}
}

// TestTradeStagingTimeoutDispatchesItsReverseWalk names the specific defect, so
// a regression reads as what it is rather than as a table row.
func TestTradeStagingTimeoutDispatchesItsReverseWalk(t *testing.T) {
	logger, _ := test.NewNullLogger()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	// See the table test above on why the shared cache is left alone.
	s, _ := NewBuilder().SetSagaType(TradeStaging).SetInitiatedBy("test").Build()

	if !dispatchTimeoutRollbacks(logger, ctx, s) {
		t.Fatal("a staging saga that times out between release_from_character and accept_to_trade rolls back nothing: compartment -1, escrow +0, item destroyed")
	}
}

// TestEverySagaTypeIsClassified closes the hole that let seven saga types carry
// the unrolled-timeout defect unnoticed.
//
// TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout iterates
// reverseWalkSagaTypes, so it can only catch a type that is IN the list but
// missing from the switch. A type nobody put in the list is invisible to it —
// which is exactly what happened: the list originally held only the four types
// the trade work touched, while seven others had a bespoke step-failure
// compensator and no timeout arm at all.
//
// This asserts the two classification lists together account for every known
// type, so adding a saga type without deciding whether it needs a reverse walk
// fails here rather than silently destroying value on a 30s backstop.
func TestEverySagaTypeIsClassified(t *testing.T) {
	classified := make(map[Type]int, len(allSagaTypes))
	for _, st := range reverseWalkSagaTypes {
		classified[st]++
	}
	for _, st := range noReverseWalkSagaTypes {
		classified[st]++
	}

	for _, st := range allSagaTypes {
		switch classified[st] {
		case 0:
			t.Errorf("saga type %q is in neither reverseWalkSagaTypes nor noReverseWalkSagaTypes; if it has a bespoke compensator it will roll back NOTHING on the 30s timeout backstop", st)
		case 1:
			// classified exactly once, which is the requirement
		default:
			t.Errorf("saga type %q is in both classification lists; the timeout behaviour it gets is whichever the switch happens to match", st)
		}
		delete(classified, st)
	}
	for st := range classified {
		t.Errorf("saga type %q is classified but absent from allSagaTypes, so nothing checks its dispatch arm", st)
	}
}
