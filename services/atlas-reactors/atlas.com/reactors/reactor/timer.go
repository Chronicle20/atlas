package reactor

import (
	"atlas-reactors/kafka/producer"
	"context"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

var (
	pendingStateTimeouts     = make(map[uint32]*time.Timer)
	pendingStateTimeoutsLock sync.Mutex
)

// scheduleStateTimeout arms a process-local timer for a reactor's current
// state if both Timeout(state) > 0 and TimeoutNextState(state) are set.
// A previously-armed timer for this reactor is cancelled before a new one is
// armed (idempotent).
//
// On fire the callback re-fetches the reactor from the registry, verifies the
// state has not changed (a hit or another transition would have cancelled the
// timer, but this guards against races), transitions to the configured next
// state, emits a TRIGGER, and re-arms if the new state also has a timer.
func scheduleStateTimeout(l logrus.FieldLogger, ctx context.Context, r Model) {
	d := r.Data()
	timeoutMs := d.Timeout(r.State())
	nextState, hasNext := d.TimeoutNextState(r.State())
	if timeoutMs <= 0 || !hasNext {
		return
	}

	reactorId := r.Id()
	t := tenant.MustFromContext(ctx)

	pendingStateTimeoutsLock.Lock()
	defer pendingStateTimeoutsLock.Unlock()

	if existing, ok := pendingStateTimeouts[reactorId]; ok {
		existing.Stop()
		delete(pendingStateTimeouts, reactorId)
	}

	delay := time.Duration(timeoutMs) * time.Millisecond
	l.Debugf("Arming state-timeout for reactor [%d] at state [%d]: %v -> state [%d].", reactorId, r.State(), delay, nextState)

	pendingStateTimeouts[reactorId] = time.AfterFunc(delay, func() {
		pendingStateTimeoutsLock.Lock()
		delete(pendingStateTimeouts, reactorId)
		pendingStateTimeoutsLock.Unlock()

		current, err := GetRegistry().Get(t, reactorId)
		if err != nil {
			l.Debugf("State-timeout fired for reactor [%d], but it no longer exists. Skipping.", reactorId)
			return
		}
		if current.State() != r.State() {
			l.Debugf("State-timeout fired for reactor [%d], but state changed [%d] -> [%d]. Skipping stale fire.", reactorId, r.State(), current.State())
			return
		}

		updated, err := GetRegistry().Update(t, reactorId, func(b *ModelBuilder) {
			b.SetState(nextState)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to update reactor [%d] state on timer fire.", reactorId)
			return
		}

		l.Debugf("Reactor [%d] timer-advanced from state [%d] to [%d].", reactorId, r.State(), nextState)

		// Re-arm for the new state BEFORE downstream emits. If Kafka is slow or down,
		// we must not block the chained timer sequence. The new state's timer must
		// be armed promptly so subsequent timeouts are not delayed.
		scheduleStateTimeout(l, ctx, updated)

		Trigger(l)(ctx)(updated, 0)

		if err := producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false)); err != nil {
			l.WithError(err).Warnf("Failed to emit HIT status event for reactor [%d] after timer fire.", reactorId)
		}
	})
}

// cancelStateTimeout stops any pending state timer for a reactor.
func cancelStateTimeout(reactorId uint32) {
	pendingStateTimeoutsLock.Lock()
	defer pendingStateTimeoutsLock.Unlock()

	if timer, ok := pendingStateTimeouts[reactorId]; ok {
		timer.Stop()
		delete(pendingStateTimeouts, reactorId)
	}
}

// cancelAllStateTimeouts stops every pending state timer. Called during
// service teardown alongside CancelAllPendingActivations.
func cancelAllStateTimeouts() {
	pendingStateTimeoutsLock.Lock()
	defer pendingStateTimeoutsLock.Unlock()

	for id, timer := range pendingStateTimeouts {
		timer.Stop()
		delete(pendingStateTimeouts, id)
	}
}
