package saga

import (
	"context"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
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
