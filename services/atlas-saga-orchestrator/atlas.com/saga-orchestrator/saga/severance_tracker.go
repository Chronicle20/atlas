package saga

import (
	"sync"

	"github.com/google/uuid"
)

// severancePair identifies one direction of a buddy-severance acknowledgement:
// characterId's own buddy list lost buddyId. sever_buddies_for_transfer
// (task-227 FR-4.3) requires BOTH directions severed for every buddy, and
// atlas-buddies' REQUEST_DELETE only removes the caller's own entry, so the
// handler emits one command per direction per buddy — 2×len(BuddyIds) total —
// and the step may only complete once every one of them has been observed
// acknowledged. A bare counter cannot express that safely under Kafka's
// at-least-once delivery: a redelivered BUDDY_REMOVED would double-count and
// complete the step early on fewer real acknowledgements than were sent. A
// set of pairs is naturally idempotent — recording the same pair twice does
// not grow the set — so redelivery is a no-op rather than a miscount.
type severancePair struct {
	CharacterId uint32
	BuddyId     uint32
}

type severanceState struct {
	mu       sync.Mutex
	expected map[severancePair]struct{}
	acked    map[severancePair]struct{}
}

// severanceTrackers holds in-flight sever_buddies_for_transfer acknowledgement
// state, keyed by transactionId. Package-level so it survives the per-request
// Processor instances, the same way unmatchedEventWarnOnce does.
var severanceTrackers sync.Map // key: uuid.UUID, value: *severanceState

// RegisterSeveranceTracker records the full set of (characterId, buddyId)
// severance pairs — one per direction per buddy in buddyIds — that a
// sever_buddies_for_transfer step must observe before it may complete. The
// caller (handleSeverBuddiesForTransfer) MUST call this before emitting any
// REQUEST_DELETE command, so a BUDDY_REMOVED ack that wins the race against
// registration is never lost to a not-yet-existing tracker.
//
// A call for a transactionId that already has a tracker is a no-op — a
// re-register (e.g. a redispatched step) must never clobber acks already
// recorded, which would make the step wait on pairs that already came back.
func RegisterSeveranceTracker(transactionId uuid.UUID, characterId uint32, buddyIds []uint32) {
	expected := make(map[severancePair]struct{}, len(buddyIds)*2)
	for _, buddyId := range buddyIds {
		expected[severancePair{CharacterId: characterId, BuddyId: buddyId}] = struct{}{}
		expected[severancePair{CharacterId: buddyId, BuddyId: characterId}] = struct{}{}
	}
	st := &severanceState{
		expected: expected,
		acked:    make(map[severancePair]struct{}, len(expected)),
	}
	severanceTrackers.LoadOrStore(transactionId, st)
}

// AcknowledgeSeverance records that characterId's buddy list lost buddyId
// (one BUDDY_REMOVED event) for transactionId, and reports whether every
// pair RegisterSeveranceTracker registered for this step has now been
// acknowledged — the only condition under which the caller may advance the
// step. An ack for an untracked transactionId (no tracker registered, an
// untracked pair, or a step that already completed and had its tracker
// cleared) safely reports false; it never produces a false positive.
func AcknowledgeSeverance(transactionId uuid.UUID, characterId uint32, buddyId uint32) bool {
	v, ok := severanceTrackers.Load(transactionId)
	if !ok {
		return false
	}
	st := v.(*severanceState)
	st.mu.Lock()
	defer st.mu.Unlock()

	pair := severancePair{CharacterId: characterId, BuddyId: buddyId}
	if _, tracked := st.expected[pair]; !tracked {
		return false
	}
	st.acked[pair] = struct{}{}
	if len(st.acked) < len(st.expected) {
		return false
	}
	severanceTrackers.Delete(transactionId)
	return true
}
