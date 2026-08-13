// Package combo owns the Aran combo counter's channel-local state.
//
// The count is deliberately NOT stored as the ARAN_COMBO buff stat: that stat
// is a damage-calculation input the client decodes as a signed short, and the
// count is delivered to the client by SHOW_COMBO instead (task-217 design.md
// §2.3, §5.1). Keeping the count here also keeps the hit-frequency increment
// path free of both Redis and Kafka.
package combo

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ComboCap bounds the count at five digits. The client's combo renderer
// decomposes m_nCombo into a 5-slot digit-layer array; a six-digit count
// would overrun it (task-217 design.md §2.6). Nothing in WZ governs a cap.
const ComboCap int32 = 99999

// DefaultIdleWindow is the fallback when a tenant's socket config carries no
// idleResetMs handler option. It matches the client's own ClearCombo timer on
// five of the six in-scope versions; v95 uses 5000 ms and configures it
// (task-217 design.md §2.5, §4).
const DefaultIdleWindow = 3000 * time.Millisecond

// Eligibility is the cached result of the server-side gate evaluation: which
// Combo Ability the character owns, at what level, and the ARAN_COMBO statup
// amount to seed the buff with.
type Eligibility struct {
	comboId    skill.Id
	comboLevel byte
	statAmount int32
}

func NewEligibility(comboId skill.Id, comboLevel byte, statAmount int32) Eligibility {
	return Eligibility{comboId: comboId, comboLevel: comboLevel, statAmount: statAmount}
}

func (e Eligibility) ComboId() skill.Id { return e.comboId }
func (e Eligibility) ComboLevel() byte  { return e.comboLevel }
func (e Eligibility) StatAmount() int32 { return e.statAmount }

// Entry is one character's live combo state.
type Entry struct {
	count       int32
	lastHit     time.Time
	window      time.Duration
	f           field.Model
	eligibility Eligibility
	checkedAt   time.Time
}

func (e Entry) Count() int32             { return e.count }
func (e Entry) LastHit() time.Time       { return e.lastHit }
func (e Entry) Window() time.Duration    { return e.window }
func (e Entry) Field() field.Model       { return e.f }
func (e Entry) Eligibility() Eligibility { return e.eligibility }
func (e Entry) CheckedAt() time.Time     { return e.checkedAt }

type bucket struct {
	t       tenant.Model
	entries map[uint32]Entry
}

// Mirror is the process-wide, tenant-keyed combo state.
//
// Process-local by design: a combo lives 3-5 seconds and dies with the
// session, so losing it to a channel restart is indistinguishable from an
// idle reset, and a session is pinned to one channel process so there is no
// cross-process reader (task-217 design.md §3.2). Same accepted degradation
// as BeaconMirror.
type Mirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]*bucket
}

var (
	mirror     *Mirror
	mirrorOnce sync.Once
)

// GetMirror returns the process-wide singleton, lazily initialising it.
func GetMirror() *Mirror {
	mirrorOnce.Do(func() { mirror = &Mirror{} })
	return mirror
}

// bucketFor returns the tenant's bucket, creating it. Callers hold m.mu.
func (m *Mirror) bucketFor(t tenant.Model) *bucket {
	if m.perTenant == nil {
		m.perTenant = make(map[uuid.UUID]*bucket)
	}
	b, ok := m.perTenant[t.Id()]
	if !ok {
		b = &bucket{t: t, entries: make(map[uint32]Entry)}
		m.perTenant[t.Id()] = b
	}
	// Refresh the stored tenant so the decay sweep always rebuilds a context
	// from a current model.
	b.t = t
	return b
}

// SetEligibility records or refreshes the character's gate result without
// touching the count. Called from the attack pipeline and from the handler's
// lazy cold-start fetch.
func (m *Mirror) SetEligibility(t tenant.Model, characterId uint32, f field.Model, e Eligibility, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.bucketFor(t)
	entry := b.entries[characterId]
	entry.eligibility = e
	entry.checkedAt = now
	entry.f = f
	b.entries[characterId] = entry
}

// Eligibility returns the cached gate result when it is present and no older
// than ttl.
func (m *Mirror) Eligibility(t tenant.Model, characterId uint32, now time.Time, ttl time.Duration) (Eligibility, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.perTenant[t.Id()]
	if !ok {
		return Eligibility{}, false
	}
	e, ok := b.entries[characterId]
	if !ok || e.checkedAt.IsZero() || now.Sub(e.checkedAt) > ttl {
		return Eligibility{}, false
	}
	return e.eligibility, true
}

// Increment advances the count by one, clamped at ComboCap, and refreshes the
// idle timer. seeded reports the 0 -> 1 transition, which is the only moment
// the Combo Ability buff needs applying.
func (m *Mirror) Increment(t tenant.Model, characterId uint32, f field.Model, window time.Duration, now time.Time) (int32, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.bucketFor(t)
	e := b.entries[characterId]
	seeded := e.count == 0
	if e.count < ComboCap {
		e.count++
	} else {
		seeded = false
	}
	e.lastHit = now
	e.window = window
	e.f = f
	b.entries[characterId] = e
	return e.count, seeded
}

// Clear drops the character's entry entirely: count, idle timer, and cached
// eligibility. Session end, map change, and the client's own combo cancel all
// funnel here.
func (m *Mirror) Clear(t tenant.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.perTenant[t.Id()]; ok {
		delete(b.entries, characterId)
	}
}

// Expired describes one combo the idle sweep zeroed.
type Expired struct {
	t           tenant.Model
	characterId uint32
	f           field.Model
	comboId     skill.Id
}

func (e Expired) Tenant() tenant.Model { return e.t }
func (e Expired) CharacterId() uint32  { return e.characterId }
func (e Expired) Field() field.Model   { return e.f }
func (e Expired) ComboId() skill.Id    { return e.comboId }

// ExpireIdle zeroes every entry whose idle window has elapsed and returns
// what it zeroed so the caller can cancel the buff. The cached eligibility is
// intentionally retained: the character is still an Aran with a polearm, and
// dropping it would force a refetch on their next hit.
//
// No packet is sent for an expiry -- SHOW_COMBO 0 cannot clear the client's
// HUD, and the client clears itself on the same schedule (design.md §5.3).
func (m *Mirror) ExpireIdle(now time.Time) []Expired {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Expired
	for _, b := range m.perTenant {
		for id, e := range b.entries {
			if e.count <= 0 || now.Sub(e.lastHit) <= e.window {
				continue
			}
			e.count = 0
			b.entries[id] = e
			out = append(out, Expired{t: b.t, characterId: id, f: e.f, comboId: e.eligibility.comboId})
		}
	}
	return out
}
