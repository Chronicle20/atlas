package craft

import (
	"sync"

	"github.com/google/uuid"
)

// inflightKey identifies one (tenant, character) craft-in-progress slot
// (design §7).
type inflightKey struct {
	tenantId    uuid.UUID
	characterId uint32
}

// inflightGuard is the design §7 per-(tenant, characterId) in-memory
// duplicate-suppression window: taken when Create begins so a second
// MAKER_SKILL arriving while one is still resolving is rejected with
// craft_in_progress, and released either immediately (a synchronous
// rejection never emitted anything to wait on) or by the saga's terminal
// event (an accepted craft, released by whatever consumes that event --
// Task 24's job to wire, via Processor.ReleaseInFlight).
//
// It is deliberately NOT durable state (design §4.2.6): a process restart
// loses every entry, degrading to the ordinary validation path, which is
// still server-authoritative and cannot double-spend a material that is no
// longer there.
type inflightGuard struct {
	mu       sync.Mutex
	inflight map[inflightKey]struct{}
}

func newInflightGuard() *inflightGuard {
	return &inflightGuard{inflight: map[inflightKey]struct{}{}}
}

// TryAcquire atomically checks and marks tenantId/characterId in flight,
// returning true only for the caller that wins the race -- the property
// TestGuardIsConcurrencySafe exercises directly.
func (g *inflightGuard) TryAcquire(tenantId uuid.UUID, characterId uint32) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	k := inflightKey{tenantId: tenantId, characterId: characterId}
	if _, ok := g.inflight[k]; ok {
		return false
	}
	g.inflight[k] = struct{}{}
	return true
}

// Release clears tenantId/characterId's in-flight mark. Releasing a key
// that is not held is a no-op, not an error, since a saga's terminal event
// consumer has no way to know whether a restart already dropped it.
func (g *inflightGuard) Release(tenantId uuid.UUID, characterId uint32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.inflight, inflightKey{tenantId: tenantId, characterId: characterId})
}

// craftGuard is the process-wide guard every Processor built by NewProcessor
// shares, mirroring recipe.recipeIndex's process-wide-cache shape: the
// suppression window's whole point is to be visible across every request
// for the same character, not scoped to one Processor instance.
var craftGuard = newInflightGuard()
