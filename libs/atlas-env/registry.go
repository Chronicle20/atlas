package env

import (
	"fmt"
	"sync"
	"time"
)

// StaleAfter is the FR-1.7 staleness bound: four missed 30s heartbeats.
const StaleAfter = 120 * time.Second

// Registry answers the four FR-1 queries from an in-memory projection of the
// environment-status topic. No implementation does I/O on the query path
// (NG4) — the projection is built once, out of band, by the Kafka consumer
// that calls Apply/ApplyTombstone/Observe.
type Registry interface {
	// EnvironmentNamespace is the environment's OWN namespace — where its
	// own ingress lives. Never falls back to the baseline.
	EnvironmentNamespace(e Id) (string, error)
	// ServiceNamespace is the namespace of the effective implementation of
	// service for e: the override's namespace if e overrides it, else the
	// baseline's namespace (FR-1.2).
	ServiceNamespace(e Id, service string) (string, error)
	EnvironmentsOwnedBy(service string) []Id // FR-1.3
	IsOwner(e Id, service string) bool       // FR-1.4
	IsActive(e Id) bool                      // FR-1.4
	Stale() bool                             // FR-1.7
}

// MapRegistry is the in-memory Registry implementation: a projection of the
// environment-status topic keyed by Id, plus a heartbeat clock for
// staleness. self identifies which deployment this process is, for IsOwner
// and EnvironmentsOwnedBy.
type MapRegistry struct {
	mu       sync.RWMutex
	self     Id
	records  map[Id]Record
	lastSeen time.Time
	now      func() time.Time
}

// NewMapRegistry constructs an empty registry for the deployment identified
// by self. clock defaults to time.Now when nil; tests inject a fake clock so
// staleness tests never sleep.
func NewMapRegistry(self Id, clock func() time.Time) *MapRegistry {
	if clock == nil {
		clock = time.Now
	}
	return &MapRegistry{self: self, records: map[Id]Record{}, now: clock}
}

// Apply projects one record into the registry. A PhaseDeleted record removes
// the entry, matching a Kafka tombstone's effect.
func (r *MapRegistry) Apply(rec Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec.Phase == PhaseDeleted {
		delete(r.records, rec.Name)
	} else {
		r.records[rec.Name] = rec
	}
	r.lastSeen = r.now()
}

// ApplyTombstone removes an environment's record, matching a Kafka
// tombstone (null value) on the environment-status topic (FR-5.7).
func (r *MapRegistry) ApplyTombstone(name Id) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, name)
	r.lastSeen = r.now()
}

// Observe records that the projection is alive at t — a heartbeat record or
// any other message on the topic.
func (r *MapRegistry) Observe(at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSeen = at
}

// Stale reports whether more than StaleAfter has elapsed since the last
// observed message. A registry that has never observed anything is not
// stale — it is legacy mode, with no records and every query answering the
// FR-1.8 legacy default.
func (r *MapRegistry) Stale() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.lastSeen.IsZero() {
		return false
	}
	return r.now().Sub(r.lastSeen) > StaleAfter
}

// EnvironmentNamespace is the environment's OWN namespace — where its own
// ingress lives. It NEVER falls back to the baseline: every environment
// deploys its own ingress, and the per-service override/baseline decision is
// made inside that ingress by its NS_* routing table (Task 43). Falling back
// here would send a baseline pod's downstream call for pr-123 into main.
func (r *MapRegistry) EnvironmentNamespace(e Id) (string, error) {
	if e == "" {
		return "", nil // legacy: caller keeps its own BASE_SERVICE_URL
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[e]
	if !ok {
		return "", fmt.Errorf("environment %q is not in the registry", e)
	}
	return rec.Namespace, nil
}

// ServiceNamespace is the namespace of the effective implementation of
// service for e: e's own namespace when e overrides the service, otherwise
// e's baseline's namespace (FR-1.2). Used to generate the ingress routing
// table and for diagnostics — NOT on the REST hot path, which wants
// EnvironmentNamespace.
func (r *MapRegistry) ServiceNamespace(e Id, service string) (string, error) {
	if e == "" {
		return "", nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[e]
	if !ok {
		return "", fmt.Errorf("environment %q is not in the registry", e)
	}
	if ns, ok := rec.Overrides[service]; ok {
		return ns, nil
	}
	base, ok := r.records[rec.Baseline]
	if !ok {
		return "", fmt.Errorf("baseline %q of environment %q is not in the registry", rec.Baseline, e)
	}
	return base.Namespace, nil
}

// IsActive reports whether e's record is present and in PhaseActive.
func (r *MapRegistry) IsActive(e Id) bool {
	if e == "" {
		return true // FR-1.8
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[e]
	return ok && rec.Active()
}

// IsOwner reports whether THIS process's deployment is the effective
// implementation of service for environment e. It is a pure function of the
// projected log plus self, so exactly one deployment satisfies it and every
// pod agrees (FR-4.6).
func (r *MapRegistry) IsOwner(e Id, service string) bool {
	if e == "" {
		return true // FR-1.8: the local deployment owns everything
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[e]
	if !ok || !rec.Active() {
		return false // D4: unknown or not-yet-active is never owned
	}
	if _, overridden := rec.Overrides[service]; overridden {
		return e == r.self
	}
	return rec.Baseline == r.self
}

// EnvironmentsOwnedBy returns every environment whose effective
// implementation of service is this process's own deployment (self). With
// no records projected yet it returns [""] — exactly today's single
// legacy iteration (FR-1.8, FR-6.6).
func (r *MapRegistry) EnvironmentsOwnedBy(service string) []Id {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.records) == 0 {
		return []Id{""}
	}
	out := make([]Id, 0, len(r.records))
	for name, rec := range r.records {
		if !rec.Active() {
			continue
		}
		if _, overridden := rec.Overrides[service]; overridden {
			if name == r.self {
				out = append(out, name)
			}
			continue
		}
		if rec.Baseline == r.self {
			out = append(out, name)
		}
	}
	return out
}

// legacyRegistry is the process-wide default before SetRegistry runs: every
// query returns the legacy answer, so a service that has not yet been
// migrated behaves exactly as it does today (FR-1.8).
type legacyRegistry struct{}

func (legacyRegistry) EnvironmentNamespace(Id) (string, error)     { return "", nil }
func (legacyRegistry) ServiceNamespace(Id, string) (string, error) { return "", nil }
func (legacyRegistry) EnvironmentsOwnedBy(string) []Id             { return []Id{""} }
func (legacyRegistry) IsOwner(Id, string) bool                     { return true }
func (legacyRegistry) IsActive(Id) bool                            { return true }
func (legacyRegistry) Stale() bool                                 { return false }

var (
	currentMu sync.RWMutex
	current   Registry = legacyRegistry{}
)

// SetRegistry installs the process-wide registry. Called once, from
// libs/atlas-service's bootstrap wiring. Never call it from a domain
// package (env-domain-guard).
func SetRegistry(r Registry) {
	currentMu.Lock()
	defer currentMu.Unlock()
	current = r
}

// CurrentRegistry is never nil: before SetRegistry it is the legacy no-op.
func CurrentRegistry() Registry {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current
}
