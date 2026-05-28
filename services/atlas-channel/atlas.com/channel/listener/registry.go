package listener

import (
	"context"
	"sync"
	"time"

	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Dependencies is the seam between this package and the rest of
// atlas-channel. The fields are function values rather than interfaces so
// tests can inject minimal stubs without producing a mock that has to
// track an evolving interface surface.
type Dependencies struct {
	// UnregisterChannel calls atlas-world's DELETE channel endpoint. A
	// 404 from upstream is success.
	UnregisterChannel func(ch channel.Model) error

	// SessionsForKey enumerates sessions currently bound to the
	// (tenant, world, channel) named by key. The returned slice is a
	// snapshot — drain iterates it without re-querying.
	SessionsForKey func(key server.Key) []Session

	// SendShutdownNotice writes the server-shutdown packet to s. Best
	// effort — failures are logged, not returned.
	SendShutdownNotice func(s Session)

	// DestroySession invokes the session.Processor.Destroy path for s.
	// Returns an error so drain can record it; the drain continues to
	// the next session either way.
	DestroySession func(s Session) error

	// RemoveHandler maps to consumer.Manager.RemoveHandler — invoked
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
// Draining/Removed state, the caller must wait — Add does not race a
// Drain to revive a terminal Handle.
func (r *Registry) Add(parent context.Context, key server.Key, sc server.Model, body func(h *Handle) ([]HandlerHandle, error)) (*Handle, error) {
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
func (r *Registry) Get(key server.Key) (*Handle, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.entries[key]
	return h, ok
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
//	         call atlas-world DELETE so new clients can't pick this
//	         channel.
//	Phase 2 (save-and-kick): enumerate sessions for key, send shutdown
//	         notice, destroy each.
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
	r.l.WithField("key", key).Info("listener.drain_phase phase=1")

	// Phase 2: save-and-kick existing sessions.
	sessions := r.deps.SessionsForKey(key)
	for _, s := range sessions {
		r.deps.SendShutdownNotice(s)
		if err := r.deps.DestroySession(s); err != nil {
			r.l.WithError(err).WithField("key", key).Warn("listener.drain.destroy_session_failed")
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

// DrainAll iterates the current snapshot and drains every Handle.
// Concurrent calls are safe; the per-Handle Drain serializes itself.
// Used on SIGTERM so the pod stops serving traffic cleanly.
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
