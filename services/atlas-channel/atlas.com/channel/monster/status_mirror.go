package monster

import (
	"sync"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// StatusEffectAppliedBody mirrors the wire payload of a monster
// STATUS_APPLIED Kafka event from atlas-monsters. The wire EffectId is a
// string (see services/atlas-monsters/.../monster/kafka.go and
// services/atlas-channel/.../kafka/message/monster/kafka.go), so we keep
// the field as string here too. Reflect* fields capture the geometric
// reflect window so the attack handler can compute reflect math without
// a Kafka round-trip per damage entry.
type StatusEffectAppliedBody struct {
	EffectId          string           `json:"effectId"`
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          int64            `json:"duration"`
	TickInterval      int64            `json:"tickInterval"`
	ReflectKind       string           `json:"reflectKind"`
	ReflectPercent    int32            `json:"reflectPercent"`
	ReflectLtX        int16            `json:"reflectLtX"`
	ReflectLtY        int16            `json:"reflectLtY"`
	ReflectRbX        int16            `json:"reflectRbX"`
	ReflectRbY        int16            `json:"reflectRbY"`
	ReflectMaxDamage  int32            `json:"reflectMaxDamage"`
}

// ReflectInfo is the projected, attack-handler-ready view of a reflect
// effect currently active on a monster.
type ReflectInfo struct {
	Kind      string
	Percent   int32
	LtX       int16
	LtY       int16
	RbX       int16
	RbY       int16
	MaxDamage int32
	ExpiresAt time.Time
}

// StatusEntry is a single status effect projected into the mirror.
// Reflect is non-nil iff the wire body carried reflect information.
type StatusEntry struct {
	EffectId  string
	Statuses  map[string]int32
	Reflect   *ReflectInfo
	ExpiresAt time.Time
}

// StatusMirror is a per-channel-process, in-memory projection of monster
// status events. The attack handler reads it on each damage entry to
// compute reflect contribution; the per-tenant nesting lets us preserve
// tenant isolation under multi-tenancy. Keyed by tenant -> uniqueId ->
// effectKey -> []StatusEntry. effectKey is a status name like "VENOM" or
// "MAGIC_REFLECT" so VenomCount and GetReflect remain O(stack-depth).
type StatusMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]map[string][]StatusEntry
}

var (
	statusMirrorOnce sync.Once
	statusMirror     *StatusMirror
)

// GetStatusMirror returns the process-wide singleton mirror, lazily
// initialising it on first call. Singleton via sync.Once is the
// established Atlas pattern (see inbox.go).
func GetStatusMirror() *StatusMirror {
	statusMirrorOnce.Do(func() {
		statusMirror = &StatusMirror{
			perTenant: map[uuid.UUID]map[uint32]map[string][]StatusEntry{},
		}
	})
	return statusMirror
}

// EvictTenant drops every monster-status entry for the given tenant.
// Invoked by listener.RegisterEvictor when the last listener for the
// tenant drains so the mirror doesn't retain status from a tenant that
// is no longer being served.
func (m *StatusMirror) EvictTenant(tid uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.perTenant, tid)
}

// keys derives the effect keys (status names) carried by a body. We
// project under each status name so reflect lookups don't have to walk
// the entire monster's effect set.
func bodyKeys(b StatusEffectAppliedBody) []string {
	if len(b.Statuses) == 0 {
		return nil
	}
	keys := make([]string, 0, len(b.Statuses))
	for k := range b.Statuses {
		keys = append(keys, k)
	}
	return keys
}

func (m *StatusMirror) ensureMonster(tid uuid.UUID, uniqueId uint32) map[string][]StatusEntry {
	tenantMap, ok := m.perTenant[tid]
	if !ok {
		tenantMap = map[uint32]map[string][]StatusEntry{}
		m.perTenant[tid] = tenantMap
	}
	monsterMap, ok := tenantMap[uniqueId]
	if !ok {
		monsterMap = map[string][]StatusEntry{}
		tenantMap[uniqueId] = monsterMap
	}
	return monsterMap
}

// OnApplied records a STATUS_APPLIED event in the mirror.
func (m *StatusMirror) OnApplied(t tenant.Model, uniqueId uint32, b StatusEffectAppliedBody, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	expiresAt := now.Add(time.Duration(b.Duration) * time.Millisecond)
	var reflect *ReflectInfo
	if b.ReflectKind != "" {
		reflect = &ReflectInfo{
			Kind:      b.ReflectKind,
			Percent:   b.ReflectPercent,
			LtX:       b.ReflectLtX,
			LtY:       b.ReflectLtY,
			RbX:       b.ReflectRbX,
			RbY:       b.ReflectRbY,
			MaxDamage: b.ReflectMaxDamage,
			ExpiresAt: expiresAt,
		}
	}
	entry := StatusEntry{
		EffectId:  b.EffectId,
		Statuses:  b.Statuses,
		Reflect:   reflect,
		ExpiresAt: expiresAt,
	}

	monsterMap := m.ensureMonster(t.Id(), uniqueId)
	for _, k := range bodyKeys(b) {
		monsterMap[k] = append(monsterMap[k], entry)
	}
}

// removeByEffectId is the shared core for OnExpired/OnCancelled — drop
// every entry across every status key whose EffectId matches.
func (m *StatusMirror) removeByEffectId(t tenant.Model, uniqueId uint32, effectId string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	monsterMap, ok := tenantMap[uniqueId]
	if !ok {
		return
	}
	for key, entries := range monsterMap {
		filtered := entries[:0]
		for _, e := range entries {
			if e.EffectId != effectId {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			delete(monsterMap, key)
		} else {
			monsterMap[key] = filtered
		}
	}
	if len(monsterMap) == 0 {
		delete(tenantMap, uniqueId)
	}
}

// OnExpired removes entries matching effectId, dispatched from a
// STATUS_EXPIRED event.
func (m *StatusMirror) OnExpired(t tenant.Model, uniqueId uint32, effectId string) {
	m.removeByEffectId(t, uniqueId, effectId)
}

// OnCancelled removes entries matching effectId, dispatched from a
// STATUS_CANCELLED event.
func (m *StatusMirror) OnCancelled(t tenant.Model, uniqueId uint32, effectId string) {
	m.removeByEffectId(t, uniqueId, effectId)
}

// OnMonsterGone clears every entry for a monster, dispatched from
// MONSTER_DESTROYED / MONSTER_KILLED to keep the mirror bounded.
func (m *StatusMirror) OnMonsterGone(t tenant.Model, uniqueId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	delete(tenantMap, uniqueId)
}

// GetReflect returns an active reflect entry for the monster matching
// the requested damage class (kind), if any. A monster can carry both
// a PHYSICAL (WEAPON_COUNTER) and a MAGICAL (MAGIC_COUNTER) reflect
// concurrently, so the attack handler must specify which class applies
// to the damage entry being resolved. Entries whose ExpiresAt has
// already passed in wall-clock time are skipped — a prune event
// normally clears them, but this guards a race between time passing
// and the expiry event arriving. Returns the first non-expired reflect
// whose Kind equals the requested kind. The attack handler calls this
// per damage entry, so it must be cheap; we walk only entries indexed
// under reflect-keyed statuses.
func (m *StatusMirror) GetReflect(t tenant.Model, uniqueId uint32, kind string) (ReflectInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return ReflectInfo{}, false
	}
	monsterMap, ok := tenantMap[uniqueId]
	if !ok {
		return ReflectInfo{}, false
	}
	now := time.Now()
	for _, entries := range monsterMap {
		for i := range entries {
			e := entries[i]
			if e.Reflect == nil {
				continue
			}
			if e.Reflect.Kind != kind {
				continue
			}
			if !now.Before(e.ExpiresAt) {
				continue
			}
			return *e.Reflect, true
		}
	}
	return ReflectInfo{}, false
}

// VenomCount returns the number of distinct VENOM stacks currently
// active on the monster. Used by the venom DOT path (FR-7).
func (m *StatusMirror) VenomCount(t tenant.Model, uniqueId uint32) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return 0
	}
	monsterMap, ok := tenantMap[uniqueId]
	if !ok {
		return 0
	}
	return len(monsterMap["VENOM"])
}
