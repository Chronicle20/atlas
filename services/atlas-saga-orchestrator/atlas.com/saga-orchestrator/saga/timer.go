package saga

import (
	"context"
	"fmt"
	"sync"
	"time"

	sagaMsg "atlas-saga-orchestrator/kafka/message/saga"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TimerRegistry tracks the per-saga timeout backstop timers introduced in
// PRD §4.1 / plan Phase 4. It lives alongside the cache rather than inside
// it so the DB-backed PostgresStore does not need to reason about in-process
// Go timers.
type TimerRegistry struct {
	mu      sync.Mutex
	entries map[uuid.UUID]*time.Timer
}

var sagaTimerRegistry = &TimerRegistry{entries: make(map[uuid.UUID]*time.Timer)}

// SagaTimers returns the singleton TimerRegistry.
func SagaTimers() *TimerRegistry { return sagaTimerRegistry }

// Schedule arms a per-saga timer. If a timer already exists for the given
// transactionId (retry / re-inject), the previous one is stopped and replaced.
// The fire callback re-wraps the tenant into a fresh context.Background() so
// it survives the consumer-scoped context that originally scheduled it.
func (r *TimerRegistry) Schedule(l logrus.FieldLogger, t tenant.Model, txId uuid.UUID, timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	r.mu.Lock()
	if old, ok := r.entries[txId]; ok {
		old.Stop()
	}
	var timer *time.Timer
	timer = time.AfterFunc(timeout, func() {
		// Self-clean the registry FIRST so subsequent observers (tests, reschedules)
		// see the timer as "fired, not pending" even if downstream emission blocks.
		r.mu.Lock()
		if current, ok := r.entries[txId]; ok && current == timer {
			delete(r.entries, txId)
		}
		r.mu.Unlock()

		ctx := tenant.WithContext(context.Background(), t)
		handleSagaTimeout(l, ctx, txId, timeout)
	})
	r.entries[txId] = timer
	r.mu.Unlock()
}

// Cancel stops and forgets the timer for a saga. Safe to call on an unknown
// transactionId (idempotent).
func (r *TimerRegistry) Cancel(txId uuid.UUID) {
	r.mu.Lock()
	if t, ok := r.entries[txId]; ok {
		t.Stop()
		delete(r.entries, txId)
	}
	r.mu.Unlock()
}

// Has reports whether a timer is currently registered for the given transactionId.
// Used primarily by tests.
func (r *TimerRegistry) Has(txId uuid.UUID) bool {
	r.mu.Lock()
	_, ok := r.entries[txId]
	r.mu.Unlock()
	return ok
}

// handleSagaTimeout is the time.AfterFunc callback body for a saga's timeout
// backstop (PRD §4.1-4.3 / plan Phase 4). It takes the terminal-state guard
// (Pending → Compensating), drives the reverse-walk rollback dispatches for
// CharacterCreation sagas, finalizes the lifecycle (Compensating → Failed),
// and emits exactly one Failed event with ErrorCodeSagaTimeout.
//
// Without the dispatch step here, a wedged CharacterCreation saga would emit
// Failed correctly but leave the character + inventory rows behind in the DB
// — see the task-002 bugfix commit for details.
func handleSagaTimeout(l logrus.FieldLogger, ctx context.Context, txId uuid.UUID, timeout time.Duration) {
	c := GetCache()
	s, ok := c.GetById(ctx, txId)
	if !ok {
		// Saga already evicted (normal terminal) — nothing to do.
		return
	}
	if !c.TryTransition(ctx, txId, SagaLifecyclePending, SagaLifecycleCompensating) {
		l.WithFields(logrus.Fields{
			"transaction_id": txId.String(),
			"saga_type":      s.SagaType(),
		}).Info("saga already terminal, timeout emission skipped")
		return
	}
	reason := fmt.Sprintf("saga exceeded timeout of %s", timeout)
	l.WithFields(logrus.Fields{
		"transaction_id":  txId.String(),
		"saga_type":       s.SagaType(),
		"timeout":         timeout.String(),
		"completed_steps": s.GetCompletedStepCount(),
	}).Warn("saga timed out, dispatching compensation + emitting Failed")

	// Dispatch the inverse commands for CharacterCreation sagas. Fire-and-forget;
	// Phase-5 delete commands (atlas-character, atlas-skills) are idempotent on
	// missing rows, so out-of-order arrival is safe.
	if s.SagaType() == CharacterCreation {
		NewCompensator(l, ctx).DispatchCharacterCreationRollbacks(s)
	}

	// Finalize the lifecycle. If someone else already took Compensating → Failed
	// (unlikely — stepCompleted(false) would have cancelled this timer), skip the
	// emit to avoid duplicates.
	if !c.TryTransition(ctx, txId, SagaLifecycleCompensating, SagaLifecycleFailed) {
		l.WithFields(logrus.Fields{
			"transaction_id": txId.String(),
		}).Info("saga already finalized by another path, timeout emission skipped")
		c.Remove(ctx, txId)
		return
	}
	c.Remove(ctx, txId)

	failedStep := ""
	if step, ok := s.GetCurrentStep(); ok {
		failedStep = step.StepId()
	}
	if err := EmitSagaFailed(l, ctx, s, sagaMsg.ErrorCodeSagaTimeout, reason, failedStep); err != nil {
		l.WithError(err).WithField("transaction_id", txId.String()).Error("failed to emit timeout saga-failed event")
	}
}
