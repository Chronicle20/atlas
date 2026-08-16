package saga

import (
	"context"
	"fmt"
	"sync"
	"time"

	sagaMsg "atlas-saga-orchestrator/kafka/message/saga"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TimerRegistry tracks the per-saga timeout backstop timers introduced in
// PRD §4.1 / plan Phase 4. It lives alongside the cache rather than inside
// it so the DB-backed PostgresStore does not need to reason about in-process
// Go timers.
type TimerRegistry struct {
	mu         sync.Mutex
	entries    map[uuid.UUID]*time.Timer
	envContext func(context.Context) context.Context
}

var sagaTimerRegistry = &TimerRegistry{entries: make(map[uuid.UUID]*time.Timer)}

// SagaTimers returns the singleton TimerRegistry.
func SagaTimers() *TimerRegistry { return sagaTimerRegistry }

// SetEnvContext wires the function that originates this pod's environment
// identity onto the tenant context a fired timer rebuilds (see Schedule's
// AfterFunc callback below). saga/ is outside env-domain-guard's permitted
// atlas-env import list, so main.go supplies this as a plain function value
// (withSelfEnvironment) rather than the package importing atlas-env itself
// -- without it, every timed-out saga's compensation dispatch and Failed
// emission would resolve to the baseline URL/topic regardless of which
// environment originally drove the saga (FR-1.8, FR-3.1/FR-3.2).
func (r *TimerRegistry) SetEnvContext(f func(context.Context) context.Context) {
	r.mu.Lock()
	r.envContext = f
	r.mu.Unlock()
}

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
		r.mu.Lock()
		ec := r.envContext
		r.mu.Unlock()
		if ec != nil {
			ctx = ec(ctx)
		}
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
	// Re-read after taking the guard. The snapshot above predates the
	// Pending → Compensating transition, so a step whose success landed in that
	// window is still Pending in it — and a reverse-walk driven from the stale
	// copy skips that step's inverse for good, because the normal completion
	// path already consumed the event and no late one will arrive. Everything
	// below (the three walks and the Failed emission's failedStep) must read the
	// post-guard copy. The saga is still cached — TryTransition just succeeded
	// against it — but keep the snapshot if it raced away rather than aborting a
	// compensation that is already committed to.
	if fresh, ok := c.GetById(ctx, txId); ok {
		s = fresh
	}

	reason := fmt.Sprintf("saga exceeded timeout of %s", timeout)
	l.WithFields(logrus.Fields{
		"transaction_id":  txId.String(),
		"saga_type":       s.SagaType(),
		"timeout":         timeout.String(),
		"completed_steps": s.GetCompletedStepCount(),
	}).Warn("saga timed out, dispatching compensation + emitting Failed")

	dispatchTimeoutRollbacks(l, ctx, s)

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

// reverseWalkSagaTypes is the set of saga types whose timeout MUST dispatch a
// reverse walk, and it is the single list that dispatchTimeoutRollbacks and its
// test both read.
//
// It exists because the routing it replaced was a chain of independent `if`
// statements, and TradeStaging was added to every other place saga types are
// enumerated but not to that chain. The consequence was silent and total: a
// staging saga whose release_from_character completed and whose accept_to_trade
// then stalled past the 30s backstop rolled back NOTHING, so the compartment
// was down one item, escrow held none, and the asset simply ceased to exist. No
// race was needed — a slow consumer sufficed.
//
// A saga type belongs here exactly when CompensateFailedStep routes it to a
// bespoke compensator, because that is what "this type has inverses worth
// walking" means. Anything with an entry there and no entry here destroys value
// on timeout while looking correct on step failure.
// The list is EVERY type CompensateFailedStep routes to a bespoke compensator.
// It was originally only the four the trade work touched, which left the other
// seven carrying the identical defect this exists to close — found by the
// plan-adherence review, not by the guard below, because a type omitted from
// the list is invisible to a test that iterates the list.
var reverseWalkSagaTypes = []Type{
	CharacterCreation,
	MtsOperation,
	TradeTransaction,
	TradeStaging,
	PetEvolution,
	ItemTagUse,
	SealingLockUse,
	IncubatorUse,
	KarmaScissorsUse,
	ExpirationExtenderUse,
	PointReset,
	NoteSend,
	SkillBookUse,
	MesoSackUse,
}

// noReverseWalkSagaTypes are the saga types that deliberately have NO reverse
// walk, each because its steps are individually compensated by the per-action
// switch in CompensateFailedStep rather than by a bespoke whole-saga walk.
//
// It exists so that reverseWalkSagaTypes and this list TOGETHER account for
// every known saga type, which is what lets TestEverySagaTypeIsClassified fail
// when a new type is added and neither list is updated. Iterating
// reverseWalkSagaTypes alone could never catch that: a type nobody listed is
// invisible to a test that walks the list. That is precisely how the seven
// types above went unnoticed until a review asked the question directly.
var noReverseWalkSagaTypes = []Type{
	InventoryTransaction,
	QuestReward,
	StorageOperation,
	CharacterRespawn,
	GachaponTransaction,
}

// allSagaTypes is every type the orchestrator knows, kept beside the two
// classification lists so adding one here without classifying it fails a test.
var allSagaTypes = []Type{
	InventoryTransaction, QuestReward, TradeTransaction, TradeStaging,
	CharacterCreation, StorageOperation, CharacterRespawn, GachaponTransaction,
	PetEvolution, ItemTagUse, SealingLockUse, IncubatorUse, ExpirationExtenderUse,
	KarmaScissorsUse, PointReset,
	MtsOperation, NoteSend, SkillBookUse, MesoSackUse,
}

// dispatchTimeoutRollbacks fires the reverse walk for a timed-out saga and
// reports whether the type had one.
//
// Fire-and-forget in every arm: the inverses are idempotent (the character
// deletes tolerate missing rows) or claimed once per step via the
// lateCompensated marker (the trade walks), so neither out-of-order arrival nor
// an overlapping step-driven compensation can double-apply.
func dispatchTimeoutRollbacks(l logrus.FieldLogger, ctx context.Context, s Saga) bool {
	c := NewCompensator(l, ctx)
	switch s.SagaType() {
	case CharacterCreation:
		c.DispatchCharacterCreationRollbacks(s)
	case MtsOperation:
		// Re-credit/debit currency, re-grant item, restore holding — the
		// dupe-safety core (task-102 §4.1).
		c.DispatchMtsOperationRollbacks(s)
	case TradeTransaction:
		// Without this a timed-out trade_settlement leaves a HALF-SWAP: the
		// completed releases/accepts stand, so one side's goods moved and the
		// other's are destroyed.
		c.DispatchTradeTransactionRollbacks(s)
	case TradeStaging:
		// Without this a timed-out stage destroys the asset outright: the
		// release from the compartment completed, the accept into escrow did
		// not, and nothing puts it back.
		c.DispatchTradeStagingRollbacks(s)
	case PetEvolution:
		c.DispatchPetEvolutionRollbacks(s)
	case ItemTagUse, SealingLockUse, IncubatorUse, ExpirationExtenderUse, KarmaScissorsUse:
		// The five share one compensator, exactly as CompensateFailedStep
		// routes them.
		c.DispatchCashItemUseRollbacks(s)
	case PointReset:
		c.DispatchPointResetRollbacks(s)
	case NoteSend:
		c.DispatchNoteSendRollbacks(s)
	case SkillBookUse:
		c.DispatchSkillBookUseRollbacks(s)
	case MesoSackUse:
		// Without this a timed-out sack use is pure loss: consume_meso_sack
		// completed, award_mesos never landed, and nothing puts the sack back.
		c.DispatchMesoSackRollbacks(s)
	default:
		return false
	}
	return true
}
