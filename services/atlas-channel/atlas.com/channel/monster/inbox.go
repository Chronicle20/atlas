package monster

import (
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// Decision is the predicted next skill atlas-monsters has chosen for a
// monster, sourced from a NEXT_SKILL_DECIDED event. Sentinel SkillId == 0
// means "do not write a skill into the next ack".
type Decision struct {
	SkillId                byte
	SkillLevel             byte
	DecidedAtMs            int64
	NextEligibleRepickAtMs int64
}

// IsSentinel reports whether the decision is the no-skill sentinel.
func (d Decision) IsSentinel() bool { return d.SkillId == 0 }

// nextSkillInbox is a per-channel-process, in-memory single-use handoff
// between atlas-monsters' picker decision events and atlas-channel's
// MoveLife handler. See docs/inbox-pattern.md for the pattern.
type nextSkillInbox struct {
	mu      sync.RWMutex
	tenants map[uuid.UUID]map[uint32]Decision
}

var (
	nextSkillInboxInst *nextSkillInbox
	nextSkillInboxOnce sync.Once
)

// InitNextSkillInbox initializes the singleton. Call once at process startup.
func InitNextSkillInbox() {
	nextSkillInboxOnce.Do(func() {
		nextSkillInboxInst = &nextSkillInbox{
			tenants: make(map[uuid.UUID]map[uint32]Decision),
		}
	})
}

// GetNextSkillInbox returns the singleton inbox. Returns nil until
// InitNextSkillInbox has been called.
func GetNextSkillInbox() *nextSkillInbox { return nextSkillInboxInst }

// Put writes (or overwrites — last-writer-wins) the decision for the given
// (tenant, uniqueId) pair.
func (r *nextSkillInbox) Put(t tenant.Model, uniqueId uint32, d Decision) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tid := t.Id()
	inner, ok := r.tenants[tid]
	if !ok {
		inner = make(map[uint32]Decision)
		r.tenants[tid] = inner
	}
	inner[uniqueId] = d
}

// TakeAndClear returns the current decision for the (tenant, uniqueId) pair
// and removes it. Subsequent reads miss until a fresh Put. Single-use serve
// semantics (PRD §FR-21).
func (r *nextSkillInbox) TakeAndClear(t tenant.Model, uniqueId uint32) (Decision, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tid := t.Id()
	inner, ok := r.tenants[tid]
	if !ok {
		return Decision{}, false
	}
	d, hit := inner[uniqueId]
	if !hit {
		return Decision{}, false
	}
	delete(inner, uniqueId)
	return d, true
}

// Evict removes the entry for the given (tenant, uniqueId) without returning
// it. Used on MONSTER_DESTROYED to keep the inbox bounded.
func (r *nextSkillInbox) Evict(t tenant.Model, uniqueId uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tid := t.Id()
	inner, ok := r.tenants[tid]
	if !ok {
		return
	}
	delete(inner, uniqueId)
}

// EvictTenant drops every entry for the given tenant. Invoked by
// listener.RegisterEvictor when the last listener for a tenant drains
// so the inbox doesn't retain dangling decisions for a tenant that
// won't be served again until reconfigured.
func (r *nextSkillInbox) EvictTenant(tid uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tenants, tid)
}
