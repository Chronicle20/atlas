package chakra

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TTL bounds a Chakra recovery window server-side.
//
// It is a SAFETY bound, not a timing model. The server does not simulate the
// window: the client opens it with a prepare packet and closes it by sending
// USE_SKILL at animation end. The client's own animation is 1500 ms (the
// `prepare` node is 15 frames x 100 ms delay, identical at v48 and v83), and
// the client writes its own outer bound of 5000 ms on the Chakra path
// (design §4.3). 5 s matches that bound and leaves headroom over 1500 ms for
// latency. Not level-dependent, not version-dependent.
const TTL = 5000 * time.Millisecond

// sweepInterval is how often the background sweeper evicts expired entries.
// Correctness does not depend on it — Get applies lazy expiry — so it only
// bounds memory.
const sweepInterval = 30 * time.Second

// Key scopes a recovery window to one character in one tenant.
type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

// Entry is the snapshot taken when the recovery window opens.
//
// X and Y are captured at prepare time from the caster's REAL skill-book
// level, so the damage path never needs an atlas-data round trip per hit
// (PRD FR-2.4 / NFR hot-path cost) and a mid-window skill-book change cannot
// desync the damage factor from the heal.
type Entry struct {
	SkillLevel byte
	X          int16 // WZ `x` — damage-taken percent (design §4.1)
	Y          int16 // WZ `y` — recovery-rate percent (design §4.1)
	StartedAt  time.Time
}

type Registry struct {
	mutex   sync.RWMutex
	entries map[Key]Entry
}

var (
	registry *Registry
	once     sync.Once
)

// GetRegistry returns the process-wide recovery-state registry.
//
// In-process is the whole view, not a shard of it: a character's socket
// session lives on exactly one atlas-channel pod, and the caster is standing
// still in one map on one channel for the entire window.
func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{entries: make(map[Key]Entry)}
	})
	return registry
}

// Start opens (or restarts) a recovery window.
func (r *Registry) Start(t tenant.Model, characterId uint32, level byte, x int16, y int16, now time.Time) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entries[Key{Tenant: t, CharacterId: characterId}] = Entry{
		SkillLevel: level,
		X:          x,
		Y:          y,
		StartedAt:  now,
	}
}

// Get returns the live recovery window, if any. An entry at or past TTL
// reads as absent (lazy expiry) regardless of whether the sweeper has run.
func (r *Registry) Get(t tenant.Model, characterId uint32, now time.Time) (Entry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.entries[Key{Tenant: t, CharacterId: characterId}]
	if !ok || now.Sub(e.StartedAt) >= TTL {
		return Entry{}, false
	}
	return e, true
}

// Clear ends a recovery window and reports whether one was open. Callers
// name the reason (damaged / moved / map change / disconnect / completed) in
// their own log line, so no reason is stored here.
func (r *Registry) Clear(t tenant.Model, characterId uint32) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	k := Key{Tenant: t, CharacterId: characterId}
	if _, ok := r.entries[k]; !ok {
		return false
	}
	delete(r.entries, k)
	return true
}

// Sweep evicts expired entries and returns how many were removed.
func (r *Registry) Sweep(now time.Time) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	n := 0
	for k, e := range r.entries {
		if now.Sub(e.StartedAt) >= TTL {
			delete(r.entries, k)
			n++
		}
	}
	return n
}

// StartSweeper runs the eviction loop until ctx is done. Spawned via
// routine.Go per tools/goroutine-guard.sh.
func (r *Registry) StartSweeper(l logrus.FieldLogger, ctx context.Context) {
	routine.Go(l, ctx, func(c context.Context) {
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-c.Done():
				return
			case <-ticker.C:
				if n := r.Sweep(time.Now()); n > 0 {
					l.Debugf("Chakra recovery sweeper evicted [%d] expired entries.", n)
				}
			}
		}
	})
}
