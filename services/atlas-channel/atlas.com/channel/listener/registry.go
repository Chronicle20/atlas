package listener

import (
	"atlas-channel/server"
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Dependencies is the seam between this package and the rest of
// atlas-channel. The fields are function values rather than interfaces so
// tests can inject minimal stubs without producing a mock that has to
// track an evolving interface surface.
type Dependencies struct {
	// UnregisterChannel calls atlas-world's DELETE channel endpoint. A
	// 404 from upstream is success.
	UnregisterChannel func(ch channel.Model) error

	// RemoveHandler maps to consumer.Manager.RemoveHandler -- invoked
	// once per HandlerHandle during phase 4.
	RemoveHandler func(topic, id string) error
}

// Session is an opaque handle on a session — listener doesn't need to
// know anything about its shape, only that the deps functions can act
// on it.
type Session any

// Config configures runtime knobs that operators want to tune per
// deployment (e.g. atlas-channel vs atlas-login).
type Config struct {
	// DrainDeadline bounds phase 3 — how long Drain waits for in-flight
	// session goroutines (h.Wg) to complete before falling through to
	// phase 4 (force-cancel). Zero means default (5s); the projection
	// apply loop clamps operator input to a 10s ceiling.
	DrainDeadline time.Duration
}

// ErrDraining is returned by Add when a Handle for the key exists but is
// mid-Drain. Add does not race a Drain to revive a terminal Handle; the
// projection apply loop retries the op on its next tick
// (task-244 design.md §4.6).
var ErrDraining = errors.New("listener: handle is draining")

// Registry is the per-process owner of all live Handles. Methods are
// safe for concurrent use; Drain is idempotent (a second call on a
// Draining or Removed key is a no-op).
type Registry struct {
	l       logrus.FieldLogger
	deps    Dependencies
	cfg     Config
	mu      sync.Mutex
	entries map[server.Key]*Handle
	// refs tracks how many active listeners exist per tenant id. When
	// the count drops to zero, registered evictors fire. Decrement
	// happens in phase 4 (after State transitions to Removed).
	refs map[uuid.UUID]int
}

// NewRegistry constructs the per-process registry.
func NewRegistry(l logrus.FieldLogger, deps Dependencies, cfg Config) *Registry {
	if cfg.DrainDeadline <= 0 {
		cfg.DrainDeadline = 5 * time.Second
	}
	const drainCeiling = 10 * time.Second
	if cfg.DrainDeadline > drainCeiling {
		cfg.DrainDeadline = drainCeiling
	}
	return &Registry{
		l:       l,
		deps:    deps,
		cfg:     cfg,
		entries: make(map[server.Key]*Handle),
		refs:    make(map[uuid.UUID]int),
	}
}

// Add inserts a new Handle for key and runs body to perform per-(t,w,c)
// startup work (server.Register, account registry init, consumer
// InitHandlers, socket service). body returns the kafka HandlerHandles
// so Drain can deregister them later.
//
// Returns the new Handle on success. If a Handle for key already exists
// and is Active, returns it (idempotent re-add). If it exists but is in
// Draining state, Add returns ErrDraining and the caller retries — Add
// does not race a Drain to revive a terminal Handle.
func (r *Registry) Add(parent context.Context, key server.Key, sc server.Model, body func(h *Handle) ([]HandlerHandle, error)) (*Handle, error) {
	r.mu.Lock()
	if existing, ok := r.entries[key]; ok {
		if existing.State == Active {
			r.mu.Unlock()
			return existing, nil
		}
		// Draining: inserting a second Handle here would let the old
		// drain's phase-4 delete(r.entries, key) remove the NEW handle and
		// decrement refs for it. Refuse; the apply loop retries.
		r.mu.Unlock()
		return nil, ErrDraining
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
//
// The returned *Handle is the same pointer Drain mutates under r.mu; do
// not read mutable fields (State in particular) off it without further
// synchronization. Use State below to observe a Handle's lifecycle state
// safely.
func (r *Registry) Get(key server.Key) (*Handle, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.entries[key]
	return h, ok
}

// State returns the current lifecycle State for key under the registry
// lock. Prefer this over reading a Handle's State field directly off a
// pointer obtained from Get/Snapshot -- Drain mutates State under r.mu,
// so an unsynchronized read of the field races it.
func (r *Registry) State(key server.Key) (State, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.entries[key]
	if !ok {
		return 0, false
	}
	return h.State, true
}

// Snapshot returns a slice copy of every Handle currently tracked,
// including those in Draining state. Safe to iterate without holding
// the registry lock.
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
//	Phase 1 (quiesce): mark Draining, deregister from server.Registry,
//	         call atlas-world DELETE, then CLOSE THE LISTENER so no new
//	         client can connect and no Accept can race phase 3's Wait.
//	Phase 2 (save-and-kick): enumerate h.Sessions(), h.Kick each one.
//	Phase 3 (deadline): wait up to cfg.DrainDeadline for h.Wg; warn on
//	         timeout.
//	Phase 4 (teardown): cancel ctx, RemoveHandler per kafka handle, mark
//	         Removed, decrement tenant ref, fire evictors if zero.
func (r *Registry) Drain(key server.Key) error {
	// Phase 1: claim the drain and quiesce upstream.
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

	server.GetRegistry().Deregister(key)
	if err := r.deps.UnregisterChannel(h.ServerModel.Channel()); err != nil {
		r.l.WithError(err).WithField("key", key).Warn("listener.drain.unregister_channel_failed")
	}
	if h.CloseListener != nil {
		if err := h.CloseListener(); err != nil && !errors.Is(err, net.ErrClosed) {
			r.l.WithError(err).WithField("key", key).Warn("listener.drain.close_listener_failed")
		}
	}
	r.l.WithField("key", key).Info("listener.drain_phase phase=1")

	// Phase 2: save-and-kick existing sessions. Kicking is what makes
	// phase 3 a real bounded wait rather than a guaranteed deadline burn:
	// Kick ends in session.Model.Disconnect(), which closes the conn so
	// the session's run() goroutine returns and releases h.Wg.
	var kicked int
	if h.Sessions != nil && h.Kick != nil {
		for _, s := range h.Sessions() {
			if err := h.Kick(s); err != nil {
				r.l.WithError(err).WithField("key", key).Warn("listener.drain.kick_session_failed")
			}
			kicked++
		}
	}
	r.l.WithField("key", key).WithField("sessions", kicked).Info("listener.drain_phase phase=2")

	// Phase 3: bounded wait on session goroutines.
	done := make(chan struct{})
	routine.Go(r.l, h.Ctx, func(_ context.Context) {
		h.Wg.Wait()
		close(done)
	})
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
		if err := r.deps.RemoveHandler(hh.Topic, hh.Id); err != nil {
			r.l.WithError(err).WithFields(logrus.Fields{
				"key":   key,
				"topic": hh.Topic,
			}).Warn("listener.drain.remove_handler_failed")
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

// DrainAll drains every Handle in the current snapshot concurrently, so
// total SIGTERM drain time is bounded by one DrainDeadline rather than
// N x DrainDeadline -- sequential drains blow past a typical
// terminationGracePeriod once phase 3 is a real wait
// (task-244 design.md §4.4). Concurrent calls are safe; each Drain
// serializes itself and touches only its own handle. Uses routine.Go so a
// panic in one handle's Drain is recovered and logged rather than killing
// the process mid-shutdown; wg.Done is deferred inside fn, ahead of
// routine.Go's own recover, so wg.Wait() still joins correctly.
func (r *Registry) DrainAll() {
	var wg sync.WaitGroup
	for _, h := range r.Snapshot() {
		wg.Add(1)
		routine.Go(r.l, h.Ctx, func(_ context.Context) {
			defer wg.Done()
			if err := r.Drain(h.Key); err != nil {
				r.l.WithError(err).WithField("key", h.Key).Warn("listener.drain_all.failed")
			}
		})
	}
	wg.Wait()
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
