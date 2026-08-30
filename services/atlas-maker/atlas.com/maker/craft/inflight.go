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

// transactionKey identifies one (tenant, saga transaction) mapping back onto
// an inflightKey, so the saga terminal-event consumer -- which carries only
// a transaction id, never a character id -- can find the entry to release
// (see Track/ReleaseByTransactionId).
type transactionKey struct {
	tenantId      uuid.UUID
	transactionId uuid.UUID
}

// inflightGuard is the design §7 per-(tenant, characterId) in-memory
// duplicate-suppression window: taken when Create begins so a second
// MAKER_SKILL arriving while one is still resolving is rejected with
// craft_in_progress, and released either immediately (a synchronous
// rejection never emitted anything to wait on) or by the saga's terminal
// event (an accepted craft, released by the kafka/consumer/saga handler via
// ReleaseInFlightByTransaction).
//
// It is deliberately NOT durable state (design §4.2.6): a process restart
// loses every entry, degrading to the ordinary validation path, which is
// still server-authoritative and cannot double-spend a material that is no
// longer there.
type inflightGuard struct {
	mu            sync.Mutex
	inflight      map[inflightKey]struct{}
	byTransaction map[transactionKey]inflightKey
}

func newInflightGuard() *inflightGuard {
	return &inflightGuard{
		inflight:      map[inflightKey]struct{}{},
		byTransaction: map[transactionKey]inflightKey{},
	}
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

// Track records transactionId against an already-acquired
// tenantId/characterId entry, once the saga built from that Create call has
// been assigned an id. The saga terminal-event consumer carries only a
// transaction id, so this is what lets ReleaseByTransactionId find the entry
// to release. Tracking against a key that is not held (already released, for
// instance concurrently by a restart-recovery path) is a no-op -- there is
// nothing left to route a later release to.
func (g *inflightGuard) Track(tenantId uuid.UUID, characterId uint32, transactionId uuid.UUID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	k := inflightKey{tenantId: tenantId, characterId: characterId}
	if _, ok := g.inflight[k]; !ok {
		return
	}
	g.byTransaction[transactionKey{tenantId: tenantId, transactionId: transactionId}] = k
}

// Release clears tenantId/characterId's in-flight mark, and any transaction
// index that had been Track-ed onto it. Releasing a key that is not held is
// a no-op, not an error, since a caller has no way to know whether a restart
// already dropped it.
func (g *inflightGuard) Release(tenantId uuid.UUID, characterId uint32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	k := inflightKey{tenantId: tenantId, characterId: characterId}
	delete(g.inflight, k)
	for tk, ik := range g.byTransaction {
		if ik == k {
			delete(g.byTransaction, tk)
		}
	}
}

// ReleaseByTransactionId clears whichever (tenantId, characterId) entry was
// Track-ed under tenantId/transactionId, the saga terminal-event consumer's
// only way to reach a guard entry. Releasing an unknown transaction id is a
// no-op: it never held a mark (a synchronous rejection never emits a saga at
// all) or it was already released.
func (g *inflightGuard) ReleaseByTransactionId(tenantId uuid.UUID, transactionId uuid.UUID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	tk := transactionKey{tenantId: tenantId, transactionId: transactionId}
	k, ok := g.byTransaction[tk]
	if !ok {
		return
	}
	delete(g.inflight, k)
	delete(g.byTransaction, tk)
}

// craftGuard is the process-wide guard every Processor built by NewProcessor
// shares, mirroring recipe.recipeIndex's process-wide-cache shape: the
// suppression window's whole point is to be visible across every request
// for the same character, not scoped to one Processor instance.
var craftGuard = newInflightGuard()

// ReleaseInFlightByTransaction clears craftGuard's entry for
// tenantId/transactionId (design §7 terminal-event release). Exported so the
// kafka/consumer/saga terminal-event handler -- which has no reason to build
// a full Processor just to release a guard it does not otherwise touch --
// can reach it directly; ProcessorImpl.ReleaseInFlight forwards to the same
// call for the Processor interface's own callers.
func ReleaseInFlightByTransaction(tenantId uuid.UUID, transactionId uuid.UUID) {
	craftGuard.ReleaseByTransactionId(tenantId, transactionId)
}

// AcquireForTest and TrackForTest expose craftGuard's TryAcquire/Track to
// other packages' tests -- kafka/consumer/saga's own tests set up a held
// entry and then assert ReleaseInFlightByTransaction (reached indirectly,
// through the terminal-event handlers) does or does not clear it. Mirrors
// data/skill/cache.go's SeedForTest precedent for reaching a process-wide
// singleton from outside its owning package. Only call from tests.
func AcquireForTest(tenantId uuid.UUID, characterId uint32) bool {
	return craftGuard.TryAcquire(tenantId, characterId)
}

func TrackForTest(tenantId uuid.UUID, characterId uint32, transactionId uuid.UUID) {
	craftGuard.Track(tenantId, characterId, transactionId)
}
