package projection

import (
	"atlas-channel/configuration"
	"atlas-channel/configuration/tenant"
	"atlas-channel/listener"
	"atlas-channel/server"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AddBody is the per-(t,w,c) startup callback main.go provides. It runs
// inside listener.Registry.Add (so it's already serialized for this key)
// and returns the kafka HandlerHandles collected from every InitHandlers
// call so Drain can deregister them later.
type AddBody func(parent context.Context, key server.Key, cfg ListenerConfig, h *listener.Handle) ([]listener.HandlerHandle, error)

// ServerModelFn builds the server.Model that the listener tracks. Passed
// in (rather than inlined) so test code can inject a stub without
// touching the real server.Register side effect.
type ServerModelFn func(key server.Key, cfg ListenerConfig) server.Model

// ApplyLoop drives listener.Registry from snapshots of the projection
// State. A single goroutine takes successive snapshots, diffs them, and
// executes Drain/Add ops in order — serialization guarantees no two
// concurrent Drain+Add races on the same key.
//
// The loop only starts producing ops once CaughtUp flips, so cold-start
// boots don't fight a half-loaded state.
type ApplyLoop struct {
	State       *State
	CaughtUp    *CaughtUp
	Registry    *listener.Registry
	AddBody     AddBody
	ServerModel ServerModelFn
	// Interval is the recheck cadence between snapshots. Defaults to
	// 250ms when zero — fast enough that an operator-driven config
	// change takes effect within a UI refresh cycle.
	Interval time.Duration
	// pending holds ops whose execution failed, keyed by the op's key, so
	// the next tick retries them. Without this a transient bind conflict
	// leaves the channel dead until config changes again: prevSvc/
	// prevTenants advance unconditionally, so ComputeOps never re-emits
	// the OpAdd (task-244 design.md §4.6).
	pending map[server.Key]Op
	// retries counts consecutive failures per pending key, so a persistent
	// conflict logs once at Warn and thereafter at Debug rather than
	// flooding at the tick cadence.
	retries map[server.Key]int
}

// Run blocks until ctx is canceled. Intended to be launched as `go
// loop.Run(ctx)` from main.go.
func (a *ApplyLoop) Run(ctx context.Context, l logrus.FieldLogger) {
	if err := a.CaughtUp.WaitCaughtUp(ctx); err != nil {
		return // ctx done before catch-up
	}
	l.Info("projection.caughtup")

	interval := a.Interval
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	a.pending = make(map[server.Key]Op)
	a.retries = make(map[server.Key]int)

	var prevSvc *configuration.RestModel
	var prevTenants map[uuid.UUID]tenant.RestModel
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			nextSvc, nextTenants := a.State.Snapshot()
			ops := ComputeOps(prevSvc, prevTenants, nextSvc, nextTenants)

			// Retry pending ops first, and only those whose key is still
			// desired -- a key that left config is dropped, not retried
			// forever.
			stillDesired := ComputeOps(nil, nil, nextSvc, nextTenants)
			desiredKeys := make(map[server.Key]bool, len(stillDesired))
			for _, op := range stillDesired {
				desiredKeys[op.Key] = true
			}
			for key, op := range a.pending {
				if !desiredKeys[key] {
					delete(a.pending, key)
					delete(a.retries, key)
					continue
				}
				if err := a.execute(ctx, l, op); err != nil {
					a.retries[key]++
					continue
				}
				delete(a.pending, key)
				delete(a.retries, key)
			}

			for _, op := range ops {
				if err := a.execute(ctx, l, op); err != nil && op.Kind == OpAdd {
					a.pending[op.Key] = op
					a.retries[op.Key] = 1
				}
			}
			prevSvc = nextSvc
			prevTenants = nextTenants
		}
	}
}

func (a *ApplyLoop) execute(ctx context.Context, l logrus.FieldLogger, op Op) error {
	switch op.Kind {
	case OpDrain:
		if err := a.Registry.Drain(op.Key); err != nil {
			l.WithError(err).WithField("key", op.Key).Warn("projection.applied drain_failed")
			return err
		}
		l.WithField("key", op.Key).WithField("op", "drain").Debug("projection.applied")
	case OpAdd:
		sc := a.ServerModel(op.Key, op.Cfg)
		_, err := a.Registry.Add(ctx, op.Key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
			// h.Ctx, NOT the apply loop's ctx: buildListener builds the
			// socket service's context from this argument, so a per-channel
			// Drain's h.Cancel() must be able to reach it. Passing the
			// apply-loop ctx here is task-244 defect 1 -- the port stayed
			// bound until full pod shutdown.
			return a.AddBody(h.Ctx, op.Key, op.Cfg, h)
		})
		if err != nil {
			// ErrDraining is expected churn (a re-add landing while the
			// old handle drains), not an operator-visible conflict -- the
			// next tick retries it either way.
			if errors.Is(err, listener.ErrDraining) {
				l.WithField("key", op.Key).WithField("retries", a.retries[op.Key]).Debug("projection.applied add_draining")
				return err
			}
			if a.retries[op.Key] > 0 {
				l.WithError(err).WithField("key", op.Key).WithField("retries", a.retries[op.Key]).Debug("projection.applied add_failed")
			} else {
				l.WithError(err).WithField("key", op.Key).Warn("projection.applied add_failed")
			}
			return err
		}
		l.WithField("key", op.Key).WithField("op", "add").Debug("projection.applied")
	}
	return nil
}
