package monster

import (
	"sync"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// AUTO_AGGRO arrives at most once per second per mob per client, from EVERY
// client that can see the mob (CMob::ApplyControl has no controller test), so
// a dense aggressive map fans in at N-clients × M-mobs per second. This gate is
// the only thing between that and a Kafka storm (design §6.3, §12).
const (
	// AutoAggroProximityThreshold is the client's own chase bar: CMob::TryFirstAttack
	// chases at |dx|/10 + |dy|/3 <= 40 (v95 @0x6482f0). A claim scoring above it is
	// dropped, which is also what lets the atlas-monsters lease expire when a player
	// walks away — the client keeps sending, but with a score out of range.
	AutoAggroProximityThreshold = uint32(40)

	// AutoAggroRefreshInterval throttles lease refreshes for an already-aggro'd mob.
	// atlas-monsters' AutoAggroLeaseTtlMs (15s) tolerates two missed refreshes.
	AutoAggroRefreshInterval = 5 * time.Second

	// autoAggroMinInterval floors the not-yet-aggro'd path. The stock client already
	// self-throttles to 1s; this guards a modified one.
	autoAggroMinInterval = 1 * time.Second

	autoAggroSweepInterval = 5 * time.Minute
	autoAggroMaxEntryAge   = 30 * time.Minute
)

// autoAggroKey identifies one character-mob claim pair.
type autoAggroKey struct {
	characterId uint32
	mobId       uint32
}

// AutoAggroGate is a per-pod, in-memory, tenant-scoped rate limiter for
// AUTO_AGGRO claims. Populated on every Admit call; evicted on tenant drain
// and a defensive staleness sweep.
type AutoAggroGate struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[autoAggroKey]time.Time
}

var (
	autoAggroGateOnce sync.Once
	autoAggroGate     *AutoAggroGate
)

// GetAutoAggroGate returns the process-wide singleton gate, lazily
// initialising it (and starting its staleness sweeper) on first call.
func GetAutoAggroGate() *AutoAggroGate {
	autoAggroGateOnce.Do(func() {
		autoAggroGate = &AutoAggroGate{perTenant: map[uuid.UUID]map[autoAggroKey]time.Time{}}
		//goroutine-guard:allow process-lifetime staleness sweeper on a sync.Once singleton with no logger/ctx in scope (GetAutoAggroGate is no-arg, called from tests and the handler path); sweepLoop only does map eviction under its own lock and cannot panic on caller input.
		go autoAggroGate.sweepLoop()
	})
	return autoAggroGate
}

func (g *AutoAggroGate) sweepLoop() {
	ticker := time.NewTicker(autoAggroSweepInterval)
	defer ticker.Stop()
	for range ticker.C {
		g.SweepStale(time.Now(), autoAggroMaxEntryAge)
	}
}

// Admit reports whether a claim for (characterId, mobId) may proceed at now,
// given whether the mob is already aggroed on this character. An already
// aggroed claim is a refresh, throttled at AutoAggroRefreshInterval; a fresh
// claim is throttled at autoAggroMinInterval. Admitting stamps now as the
// claim's last-admitted time; a blocked claim leaves the existing stamp
// untouched.
func (g *AutoAggroGate) Admit(t tenant.Model, characterId uint32, mobId uint32, aggroed bool, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	interval := autoAggroMinInterval
	if aggroed {
		interval = AutoAggroRefreshInterval
	}

	key := autoAggroKey{characterId: characterId, mobId: mobId}
	tenantMap, ok := g.perTenant[t.Id()]
	if !ok {
		tenantMap = map[autoAggroKey]time.Time{}
		g.perTenant[t.Id()] = tenantMap
	}

	if stamp, ok := tenantMap[key]; ok && now.Sub(stamp) < interval {
		return false
	}

	tenantMap[key] = now
	return true
}

// EvictTenant drops every entry for the tenant. Invoked by
// listener.RegisterEvictor when the last listener for the tenant drains.
func (g *AutoAggroGate) EvictTenant(tid uuid.UUID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.perTenant, tid)
}

// SweepStale evicts entries whose last-admitted stamp is older than maxAge
// relative to now, returning the number evicted. Exposed for tests;
// production runs it from the sweeper ticker.
func (g *AutoAggroGate) SweepStale(now time.Time, maxAge time.Duration) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	evicted := 0
	for tid, tenantMap := range g.perTenant {
		for key, stamp := range tenantMap {
			if now.Sub(stamp) > maxAge {
				delete(tenantMap, key)
				evicted++
			}
		}
		if len(tenantMap) == 0 {
			delete(g.perTenant, tid)
		}
	}
	return evicted
}
