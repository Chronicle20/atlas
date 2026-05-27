package listener

import (
	"context"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Dependencies is the seam between this package and the rest of
// atlas-login. The fields are function values rather than interfaces so
// tests can inject minimal stubs without producing a mock that has to
// track an evolving interface surface.
//
// atlas-login's Dependencies is symmetric with atlas-channel's but a few
// of the channel-only fields are omitted: there is no upstream
// UnregisterChannel call (no atlas-world DELETE), and SessionsForKey /
// SendShutdownNotice / DestroySession default to no-ops because login
// sessions are stateless after handshake and have no per-key index to
// drive a save-and-kick phase. The fields are kept so the API matches
// atlas-channel's; main.go supplies no-op closures when it doesn't have
// anything to wire.
type Dependencies struct {
	// SessionsForKey enumerates sessions currently bound to the tenant
	// named by key. The returned slice is a snapshot — drain iterates it
	// without re-querying. Login may leave this nil/empty.
	SessionsForKey func(key Key) []Session

	// SendShutdownNotice writes the server-shutdown packet to s. Best
	// effort — failures are logged, not returned. Login may leave nil.
	SendShutdownNotice func(s Session)

	// DestroySession invokes the session.Processor.Destroy path for s.
	// Returns an error so drain can record it; the drain continues to the
	// next session either way. Login may leave nil.
	DestroySession func(s Session) error

	// RemoveHandler maps to consumer.Manager.RemoveHandler — invoked once
	// per HandlerHandle during phase 4.
	RemoveHandler func(topic, id string) error
}

// Session is an opaque handle on a session — listener doesn't need to
// know anything about its shape, only that the deps functions can act on
// it.
type Session any

// Config configures runtime knobs that operators want to tune per
// deployment.
type Config struct {
	// DrainDeadline bounds phase 3 — how long Drain waits for in-flight
	// session goroutines (h.Wg) to complete before falling through to
	// phase 4 (force-cancel). Zero means default (2s for atlas-login —
	// login sessions are stateless after handshake so there's nothing to
	// flush). The projection apply loop clamps operator input to a 5s
	// ceiling (vs atlas-channel's 10s) for the same reason.
	DrainDeadline time.Duration
}

// Registry is the per-process owner of all live Handles. Methods are safe
// for concurrent use; Drain is idempotent (a second call on a Draining or
// Removed key is a no-op).
type Registry struct {
	l       logrus.FieldLogger
	deps    Dependencies
	cfg     Config
	mu      sync.Mutex
	entries map[Key]*Handle
	// refs tracks how many active listeners exist per tenant id. When the
	// count drops to zero, registered evictors fire. Decrement happens in
	// phase 4 (after State transitions to Removed). For atlas-login the
	// ref count is always 0 or 1 today (one listener per tenant), but the
	// machinery is preserved for symmetry with atlas-channel.
	refs map[uuid.UUID]int
}

// NewRegistry constructs the per-process registry. Default DrainDeadline
// is 2s (atlas-login is stateless) and the ceiling is 5s.
func NewRegistry(l logrus.FieldLogger, deps Dependencies, cfg Config) *Registry {
	if cfg.DrainDeadline <= 0 {
		cfg.DrainDeadline = 2 * time.Second
	}
	const drainCeiling = 5 * time.Second
	if cfg.DrainDeadline > drainCeiling {
		cfg.DrainDeadline = drainCeiling
	}
	return &Registry{
		l:       l,
		deps:    deps,
		cfg:     cfg,
		entries: make(map[Key]*Handle),
		refs:    make(map[uuid.UUID]int),
	}
}

// Add inserts a new Handle for key and runs body to perform per-tenant
// startup work (account registry init, consumer InitHandlers, socket
// service). body returns the kafka HandlerHandles so Drain can deregister
// them later.
//
// Returns the new Handle on success. If a Handle for key already exists
// and is Active, returns it (idempotent re-add). If it exists but is in
// Draining/Removed state, the caller must wait — Add does not race a
// Drain to revive a terminal Handle.
func (r *Registry) Add(parent context.Context, key Key, sc ServerModel, body func(h *Handle) ([]HandlerHandle, error)) (*Handle, error) {
	r.mu.Lock()
	if existing, ok := r.entries[key]; ok && existing.State == Active {
		r.mu.Unlock()
		return existing, nil
	}
	ctx, cancel := context.WithCancel(parent)
	h := &Handle{
		Key:         key,
		State:       Active,
		Ctx:         ctx,
		Cancel:      cancel,
		Wg:          &sync.WaitGroup{},
		ServerModel: sc,
	}
	r.entries[key] = h
	r.refs[key.TenantId]++
	r.mu.Unlock()

	handlers, err := body(h)
	if err != nil {
		// Rollback: body failed before the handle was usable.
		r.mu.Lock()
		delete(r.entries, key)
		r.refs[key.TenantId]--
		if r.refs[key.TenantId] <= 0 {
			delete(r.refs, key.TenantId)
		}
		r.mu.Unlock()
		cancel()
		return nil, err
	}

	r.mu.Lock()
	h.KafkaHandlers = handlers
	r.mu.Unlock()
	r.l.WithField("key", key).Info("listener.added")
	return h, nil
}

// Get returns the live Handle for key. Useful for projection diff loops
// that need to confirm whether a key is already known.
func (r *Registry) Get(key Key) (*Handle, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.entries[key]
	return h, ok
}

// Snapshot returns a slice copy of every Handle currently tracked,
// including those in Draining state. Safe to iterate without holding the
// registry lock.
func (r *Registry) Snapshot() []*Handle {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Handle, 0, len(r.entries))
	for _, h := range r.entries {
		out = append(out, h)
	}
	return out
}

// Drain runs the four-phase drain for key. Idempotent: concurrent calls
// from the projection apply loop and SIGTERM handler are safe.
//
//	Phase 1 (quiesce): mark Draining. atlas-login has no upstream to
//	         deregister from, so phase 1 is just the state transition.
//	Phase 2 (save-and-kick): enumerate sessions for key, send shutdown
//	         notice, destroy each. atlas-login leaves SessionsForKey nil
//	         by default; the loop is preserved so future work can plug
//	         it in without changing the surface.
//	Phase 3 (deadline): wait up to cfg.DrainDeadline for h.Wg; warn on
//	         timeout.
//	Phase 4 (teardown): cancel ctx, RemoveHandler per kafka handle, mark
//	         Removed, decrement tenant ref, fire evictors if zero.
func (r *Registry) Drain(key Key) error {
	// Phase 1: claim the drain (no upstream deregister for login).
	r.mu.Lock()
	h, ok := r.entries[key]
	if !ok || h.State == Removed {
		r.mu.Unlock()
		return nil
	}
	if h.State == Draining {
		r.mu.Unlock()
		return nil
	}
	h.State = Draining
	r.mu.Unlock()
	r.l.WithField("key", key).Info("listener.drain_phase phase=1")

	// Phase 2: save-and-kick existing sessions. Login leaves this empty
	// by default (sessions are stateless after handshake), but the loop
	// is preserved for symmetry with atlas-channel.
	var sessions []Session
	if r.deps.SessionsForKey != nil {
		sessions = r.deps.SessionsForKey(key)
	}
	for _, s := range sessions {
		if r.deps.SendShutdownNotice != nil {
			r.deps.SendShutdownNotice(s)
		}
		if r.deps.DestroySession != nil {
			if err := r.deps.DestroySession(s); err != nil {
				r.l.WithError(err).WithField("key", key).Warn("listener.drain.destroy_session_failed")
			}
		}
	}
	r.l.WithField("key", key).WithField("sessions", len(sessions)).Info("listener.drain_phase phase=2")

	// Phase 3: bounded wait on session goroutines.
	done := make(chan struct{})
	go func() {
		h.Wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(r.cfg.DrainDeadline):
		r.l.WithField("key", key).Warn("listener.drain_timeout")
	}
	r.l.WithField("key", key).Info("listener.drain_phase phase=3")

	// Phase 4: cancel + deregister kafka handlers, transition to Removed,
	// decrement tenant ref and fire evictors if last.
	h.Cancel()
	for _, hh := range h.KafkaHandlers {
		if r.deps.RemoveHandler != nil {
			if err := r.deps.RemoveHandler(hh.Topic, hh.Id); err != nil {
				r.l.WithError(err).WithFields(logrus.Fields{
					"key":   key,
					"topic": hh.Topic,
				}).Warn("listener.drain.remove_handler_failed")
			}
		}
	}
	r.mu.Lock()
	h.State = Removed
	delete(r.entries, key)
	r.refs[key.TenantId]--
	tenantNowEmpty := r.refs[key.TenantId] <= 0
	if tenantNowEmpty {
		delete(r.refs, key.TenantId)
	}
	r.mu.Unlock()
	r.l.WithField("key", key).Info("listener.drain_phase phase=4")

	if tenantNowEmpty {
		fireEvictors(r.l, key.TenantId)
	}
	return nil
}

// DrainAll iterates the current snapshot and drains every Handle.
// Concurrent calls are safe; the per-Handle Drain serializes itself. Used
// on SIGTERM so the pod stops serving traffic cleanly.
func (r *Registry) DrainAll() {
	for _, h := range r.Snapshot() {
		if err := r.Drain(h.Key); err != nil {
			r.l.WithError(err).WithField("key", h.Key).Warn("listener.drain_all.failed")
		}
	}
}

// fireEvictors is a package-level shim around evict.go so the registry
// doesn't need to import tenant to look up the tenant.Model — the
// callback only needs the uuid.UUID.
func fireEvictors(l logrus.FieldLogger, tenantId uuid.UUID) {
	tm, err := tenant.Create(tenantId, "", 0, 0)
	if err != nil {
		l.WithError(err).WithField("tenant", tenantId).Warn("listener.evict.tenant_synth_failed")
		return
	}
	fireEvictorsForTenant(tm)
}
