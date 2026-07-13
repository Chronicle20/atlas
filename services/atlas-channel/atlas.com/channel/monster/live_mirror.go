package monster

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// Sweep policy is leak insurance, not a tuning surface — constants, not env
// (design §3 OQ3). Eviction of a still-live entry is harmless: the next move
// takes one REST fallback and re-backfills.
const (
	liveMirrorSweepInterval = 5 * time.Minute
	liveMirrorMaxEntryAge   = 30 * time.Minute
)

// LiveEntry is the projected live state of one monster — exactly the field
// set the movement path reads (movement/processor.go), plus MaxMp so the
// PS-1 attack path can adopt the mirror without redesign.
type LiveEntry struct {
	Field              field.Model
	MonsterId          uint32
	Mp                 uint32
	MaxMp              uint32
	ControllerHasAggro bool
	LastWrite          time.Time
}

// LiveMirror is a per-pod, in-memory, tenant-scoped projection of live
// monsters, keyed by monster object id. Populated by the CREATED consumer
// handler's REST fetch and the movement path's fallback backfill; updated by
// monster_status_event events; evicted on DESTROYED/KILLED, tenant drain,
// and a defensive staleness sweep.
type LiveMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]LiveEntry
}

var (
	liveMirrorOnce sync.Once
	liveMirror     *LiveMirror
)

// GetLiveMirror returns the process-wide singleton mirror, lazily
// initialising it (and starting its staleness sweeper) on first call.
func GetLiveMirror() *LiveMirror {
	liveMirrorOnce.Do(func() {
		liveMirror = &LiveMirror{perTenant: map[uuid.UUID]map[uint32]LiveEntry{}}
		//goroutine-guard:allow process-lifetime staleness sweeper on a sync.Once singleton with no logger/ctx in scope (GetLiveMirror is no-arg, called from ~30 sites incl. tests); sweepLoop only does map eviction under its own lock and cannot panic on caller input.
		go liveMirror.sweepLoop()
	})
	return liveMirror
}

func (m *LiveMirror) sweepLoop() {
	ticker := time.NewTicker(liveMirrorSweepInterval)
	defer ticker.Stop()
	for range ticker.C {
		m.SweepStale(time.Now(), liveMirrorMaxEntryAge)
	}
}

// LiveEntryFromModel projects a full monster Model (REST shape) into a
// mirror entry. LastWrite is stamped by Put.
func LiveEntryFromModel(mo Model) LiveEntry {
	return LiveEntry{
		Field:              mo.Field(),
		MonsterId:          mo.MonsterId(),
		Mp:                 mo.Mp(),
		MaxMp:              mo.MaxMp(),
		ControllerHasAggro: mo.ControllerHasAggro(),
	}
}

// Lookup returns the entry for uniqueId, recording a hit/miss metric.
func (m *LiveMirror) Lookup(t tenant.Model, uniqueId uint32) (LiveEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		recordMirrorMiss(t)
		return LiveEntry{}, false
	}
	e, ok := tenantMap[uniqueId]
	if !ok {
		recordMirrorMiss(t)
		return LiveEntry{}, false
	}
	recordMirrorHit(t)
	return e, true
}

// Put stores a full, authoritative entry (CREATED seed or fallback backfill).
func (m *LiveMirror) Put(t tenant.Model, uniqueId uint32, e LiveEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		tenantMap = map[uint32]LiveEntry{}
		m.perTenant[t.Id()] = tenantMap
	}
	e.LastWrite = time.Now()
	tenantMap[uniqueId] = e
}

// UpdateMp sets the entry's MP to the absolute post-mutation value. Update
// only: events must never create entries, because the event envelope cannot
// supply ControllerHasAggro/MaxMp — a partial entry with a defaulted-false
// aggro flag would make the client render the mob idle. An absent entry is
// created authoritatively by the movement fallback instead (design §5.1).
func (m *LiveMirror) UpdateMp(t tenant.Model, uniqueId uint32, mpAfter uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	e, ok := tenantMap[uniqueId]
	if !ok {
		return
	}
	e.Mp = mpAfter
	e.LastWrite = time.Now()
	tenantMap[uniqueId] = e
}

// UpdateAggro sets the entry's controller-aggro flag. Update only — see
// UpdateMp for why events never create entries.
func (m *LiveMirror) UpdateAggro(t tenant.Model, uniqueId uint32, aggro bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	e, ok := tenantMap[uniqueId]
	if !ok {
		return
	}
	e.ControllerHasAggro = aggro
	e.LastWrite = time.Now()
	tenantMap[uniqueId] = e
}

// Remove evicts one monster (DESTROYED/KILLED).
func (m *LiveMirror) Remove(t tenant.Model, uniqueId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	delete(tenantMap, uniqueId)
}

// EvictTenant drops every entry for the tenant. Invoked by
// listener.RegisterEvictor when the last listener for the tenant drains.
func (m *LiveMirror) EvictTenant(tid uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.perTenant, tid)
}

// SweepStale evicts entries whose LastWrite is older than maxAge relative
// to now, returning the number evicted. Exposed for tests; production runs
// it from the sweeper ticker.
func (m *LiveMirror) SweepStale(now time.Time, maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	evicted := 0
	for tid, tenantMap := range m.perTenant {
		for id, e := range tenantMap {
			if now.Sub(e.LastWrite) > maxAge {
				delete(tenantMap, id)
				evicted++
			}
		}
		if len(tenantMap) == 0 {
			delete(m.perTenant, tid)
		}
	}
	return evicted
}
