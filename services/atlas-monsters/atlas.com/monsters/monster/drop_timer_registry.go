package monster

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas-tenant"
)

type DropTimerEntry struct {
	monsterId    uint32
	field        field.Model
	dropPeriod   time.Duration
	weaponAttack uint32
	maxHp        uint32
	lastDropAt   time.Time
	lastHitAt    time.Time
}

func (e DropTimerEntry) MonsterId() uint32        { return e.monsterId }
func (e DropTimerEntry) Field() field.Model        { return e.field }
func (e DropTimerEntry) DropPeriod() time.Duration { return e.dropPeriod }
func (e DropTimerEntry) WeaponAttack() uint32      { return e.weaponAttack }
func (e DropTimerEntry) MaxHp() uint32             { return e.maxHp }
func (e DropTimerEntry) LastDropAt() time.Time     { return e.lastDropAt }
func (e DropTimerEntry) LastHitAt() time.Time      { return e.lastHitAt }

type DropTimerRegistry struct {
	mutex   sync.RWMutex
	entries map[MonsterKey]DropTimerEntry
}

var dropTimerRegistry *DropTimerRegistry
var dropTimerOnce sync.Once

func GetDropTimerRegistry() *DropTimerRegistry {
	dropTimerOnce.Do(func() {
		dropTimerRegistry = &DropTimerRegistry{
			entries: make(map[MonsterKey]DropTimerEntry),
		}
	})
	return dropTimerRegistry
}

func (r *DropTimerRegistry) Register(t tenant.Model, uniqueId uint32, e DropTimerEntry) {
	key := MonsterKey{Tenant: t, MonsterId: uniqueId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entries[key] = e
}

func (r *DropTimerRegistry) Unregister(t tenant.Model, uniqueId uint32) {
	key := MonsterKey{Tenant: t, MonsterId: uniqueId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.entries, key)
}

func (r *DropTimerRegistry) RecordHit(t tenant.Model, uniqueId uint32, hitTime time.Time) {
	key := MonsterKey{Tenant: t, MonsterId: uniqueId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if e, ok := r.entries[key]; ok {
		e.lastHitAt = hitTime
		r.entries[key] = e
	}
}

func (r *DropTimerRegistry) UpdateLastDrop(t tenant.Model, uniqueId uint32, dropTime time.Time) {
	key := MonsterKey{Tenant: t, MonsterId: uniqueId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if e, ok := r.entries[key]; ok {
		e.lastDropAt = dropTime
		r.entries[key] = e
	}
}

func (r *DropTimerRegistry) GetAll() map[MonsterKey]DropTimerEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	result := make(map[MonsterKey]DropTimerEntry, len(r.entries))
	for k, v := range r.entries {
		result[k] = v
	}
	return result
}
