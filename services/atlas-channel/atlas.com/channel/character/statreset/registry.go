// Package statreset rate-limits the serverbound CANCEL_DEBUFF nudge
// (CWvsContext::CheckTemporaryStatDuration) per (tenant, character).
//
// Why the server must bound this independently: every client gates on
// `tick - m_tLastStatResetRequest > 200`, but v72, v79, v83, v84 and v87 never
// assign that anchor anywhere in the function. On those five the guard latches
// open 200ms after the last temporary-stat change and the client then sends
// once per frame, indefinitely — the ~30ms spacing and ~1,500 packets measured
// live on GMS 83.1. The client's 200ms floor is advisory only. (task-190 NFR-2)
//
// Why in-process state is correct rather than partial: a character's socket
// session lives on exactly one atlas-channel pod, so a per-pod map is the whole
// view, not a shard of it. On reconnect to a different pod the entry is absent
// and the first nudge passes — which is the desired behaviour anyway.
package statreset

import (
	"sync"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Window is the minimum spacing between honoured nudges from one character.
//
// 1s caps a wedged or hostile client at one Kafka command per second — a ~30x
// reduction against the measured live rate — while still recovering 10x faster
// than the 10s fleet expiry sweep (atlas-buffs tasks/expiration.go). It is a
// const, not env-configurable: a tunable would add deployment surface for a
// value with no known reason to vary per tenant. Deliberate non-goal; revisit
// only if a real workload disagrees.
const Window = 1000 * time.Millisecond

type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

type Registry struct {
	mutex sync.RWMutex
	last  map[Key]time.Time
}

var (
	registry *Registry
	once     sync.Once
)

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.last = make(map[Key]time.Time)
	})
	return registry
}

// Allow reports whether this nudge should be honoured, recording the timestamp
// when it is. The first nudge after a quiet period always passes, so recovery
// latency is one packet rather than one window.
func (r *Registry) Allow(t tenant.Model, characterId uint32, now time.Time) bool {
	k := Key{Tenant: t, CharacterId: characterId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if prev, ok := r.last[k]; ok && now.Sub(prev) < Window {
		return false
	}
	r.last[k] = now
	return true
}

// ClearCharacter drops the throttle entry for a character (session destroy).
func (r *Registry) ClearCharacter(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.last, Key{Tenant: t, CharacterId: characterId})
}
