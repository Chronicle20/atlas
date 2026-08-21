// Package listener owns the per-(tenant, world, channel) listener
// lifecycle in atlas-channel. Each Handle wraps the per-(t,w,c) startup
// work (server.Register, account registry init, consumer InitHandlers,
// socket service) and exposes a four-phase Drain so the projection apply
// loop can remove a listener cleanly when config drops it.
package listener

import (
	"atlas-channel/server"
	"context"
	"sync"
)

// State tracks where a Handle is in its lifecycle. State transitions are
// monotonic — once Removed, the Handle is not revivable; a fresh Add must
// be invoked under a new Handle.
type State int

const (
	// Active is the steady state: the listener is accepting traffic.
	Active State = iota
	// Draining means Drain has begun but teardown is in flight.
	Draining
	// Removed is terminal — all kafka handlers deregistered, ctx canceled.
	Removed
)

// HandlerHandle identifies a registered kafka consumer handler. Returned
// by InitHandlers (post Phase H sweep) and stored on Handle so Drain
// can call consumer.Manager.RemoveHandler for each.
//
// Registry.Add's body callback (projection.AddBody in main.go's
// buildListener) MUST return every HandlerHandle it has already
// registered even when it also returns a non-nil error -- e.g. a bind
// failure partway through startup, after ~20 kafka consumer handlers are
// already live. Add's rollback deregisters exactly the handles body
// returns; a nil slice on error means Add believes nothing was
// registered and leaks whatever body registered before the failure.
type HandlerHandle struct {
	Topic string
	Id    string
}

// Handle is the per-(t,w,c) listener state.
//
// CloseListener/Sessions/Kick are populated by the Add body (main.go's
// buildListener) outside r.mu. That is safe because Registry.Add
// re-acquires r.mu after body returns and every Drain reads the handle
// only after acquiring r.mu -- the lock supplies the happens-before
// edge. Do not remove that post-body lock acquisition.
type Handle struct {
	Key    server.Key
	State  State
	Ctx    context.Context
	Cancel context.CancelFunc

	// Wg tracks per-connection session goroutines (atlas-socket's run())
	// for this handle, and nothing else. It deliberately does NOT cover
	// the accept-loop goroutine -- that runs for the handle's whole
	// Active lifetime, so counting it would make drain phase 3 burn its
	// full deadline every time. It also does not cover the per-packet
	// handle() goroutines or the per-session ctx-watcher, neither of
	// which is tracked by any waitgroup today. Phase 3 therefore waits
	// on sessions, not on all in-flight work (task-244 design.md §2.2).
	Wg *sync.WaitGroup

	ServerModel   server.Model
	KafkaHandlers []HandlerHandle

	// CloseListener closes the handle's bound TCP listener. Invoked at
	// the end of drain phase 1 so the port stops accepting before the
	// phase-3 wait -- otherwise a newly accepted connection's Add(1)
	// races h.Wg.Wait() and panics with "WaitGroup misuse"
	// (task-244 design.md §2.3). nil for handles whose body never bound
	// (tests). Safe to call twice: atlas-socket's Serve guards
	// net.ErrClosed and phase 4's ctx cancellation closes it again.
	CloseListener func() error

	// Sessions snapshots the sessions bound to this handle. nil means
	// phase 2 has nothing to enumerate.
	Sessions func() []Session

	// Kick sends the shutdown notice to s and destroys it, closing the
	// underlying conn so the session's run() goroutine exits and
	// releases Wg. nil means phase 2 kicks nobody.
	Kick func(s Session) error
}
