package projection

import (
	"context"
	"time"

	"atlas-channel/configuration"
	"atlas-channel/configuration/tenant"
	"atlas-channel/listener"
	"atlas-channel/server"

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

	var prevSvc *configuration.RestModel
	var prevTenants map[uuid.UUID]tenant.RestModel
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			nextSvc, nextTenants := a.State.Snapshot()
			ops := ComputeOps(prevSvc, prevTenants, nextSvc, nextTenants)
			for _, op := range ops {
				a.execute(ctx, l, op)
			}
			prevSvc = nextSvc
			prevTenants = nextTenants
		}
	}
}

func (a *ApplyLoop) execute(ctx context.Context, l logrus.FieldLogger, op Op) {
	switch op.Kind {
	case OpDrain:
		if err := a.Registry.Drain(op.Key); err != nil {
			l.WithError(err).WithField("key", op.Key).Warn("projection.applied drain_failed")
			return
		}
		l.WithField("key", op.Key).WithField("op", "drain").Debug("projection.applied")
	case OpAdd:
		sc := a.ServerModel(op.Key, op.Cfg)
		_, err := a.Registry.Add(ctx, op.Key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
			return a.AddBody(ctx, op.Key, op.Cfg, h)
		})
		if err != nil {
			l.WithError(err).WithField("key", op.Key).Warn("projection.applied add_failed")
			return
		}
		l.WithField("key", op.Key).WithField("op", "add").Debug("projection.applied")
	}
}
