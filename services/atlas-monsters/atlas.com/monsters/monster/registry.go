package monster

import (
	"errors"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type Registry struct {
	mutex sync.Mutex

	idAllocators  map[tenant.Model]*TenantIdAllocator
	mapMonsterReg map[MapKey][]MonsterKey
	mapLocks      map[MapKey]*sync.RWMutex

	monsterReg  map[MonsterKey]Model
	monsterLock *sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetMonsterRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}

		registry.idAllocators = make(map[tenant.Model]*TenantIdAllocator)
		registry.mapMonsterReg = make(map[MapKey][]MonsterKey)
		registry.mapLocks = make(map[MapKey]*sync.RWMutex)

		registry.monsterReg = make(map[MonsterKey]Model)
		registry.monsterLock = &sync.RWMutex{}
	})
	return registry
}

func (r *Registry) getMapLock(key MapKey) *sync.RWMutex {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if val, ok := r.mapLocks[key]; ok {
		return val
	}
	var cm = &sync.RWMutex{}
	r.mapLocks[key] = cm
	r.mapMonsterReg[key] = make([]MonsterKey, 0)
	return cm
}

func (r *Registry) getOrCreateAllocator(t tenant.Model) *TenantIdAllocator {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if allocator, ok := r.idAllocators[t]; ok {
		return allocator
	}
	allocator := NewTenantIdAllocator()
	r.idAllocators[t] = allocator
	return allocator
}

func (r *Registry) CreateMonster(tenant tenant.Model, f field.Model, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) Model {
	mapKey := NewMapKey(tenant, f)

	mapLock := r.getMapLock(mapKey)
	mapLock.Lock()
	defer mapLock.Unlock()

	uniqueId := r.getOrCreateAllocator(tenant).Allocate()

	m := NewMonster(f, uniqueId, monsterId, x, y, fh, stance, team, hp, mp)

	monKey := MonsterKey{Tenant: tenant, MonsterId: m.UniqueId()}
	r.mapMonsterReg[mapKey] = append(r.mapMonsterReg[mapKey], monKey)

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	r.monsterReg[monKey] = m
	return m
}

func (r *Registry) GetMonster(tenant tenant.Model, uniqueId uint32) (Model, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}
	r.monsterLock.RLock()
	defer r.monsterLock.RUnlock()

	if m, ok := r.monsterReg[monKey]; ok {
		return m, nil
	}
	return Model{}, errors.New("monster not found")
}

func (r *Registry) GetMonstersInMap(tenant tenant.Model, f field.Model) []Model {
	mapKey := NewMapKey(tenant, f)
	mapLock := r.getMapLock(mapKey)
	mapLock.RLock()
	defer mapLock.RUnlock()

	var result []Model
	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()
	for _, monKey := range r.mapMonsterReg[mapKey] {
		if m, ok := r.monsterReg[monKey]; ok {
			result = append(result, m)
		}
	}
	return result
}

func (r *Registry) MoveMonster(tenant tenant.Model, uniqueId uint32, endX int16, endY int16, stance byte) Model {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.Move(endX, endY, stance)
		r.monsterReg[monKey] = m
		return m
	}
	return Model{}
}

func (r *Registry) ControlMonster(tenant tenant.Model, uniqueId uint32, characterId uint32) (Model, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.Control(characterId)
		r.monsterReg[monKey] = m
		return m, nil
	} else {
		return Model{}, errors.New("monster not found")
	}
}

func (r *Registry) ClearControl(tenant tenant.Model, uniqueId uint32) (Model, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.ClearControl()
		r.monsterReg[monKey] = m
		return m, nil
	} else {
		return Model{}, errors.New("monster not found")
	}
}

func (r *Registry) ApplyDamage(tenant tenant.Model, characterId uint32, damage uint32, uniqueId uint32) (DamageSummary, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.Damage(characterId, damage)
		r.monsterReg[monKey] = m
		return DamageSummary{
			CharacterId:   characterId,
			Monster:       m,
			VisibleDamage: damage,
			ActualDamage:  int64(m.Hp() - m.Hp()),
			Killed:        m.Hp() == 0,
		}, nil
	} else {
		return DamageSummary{}, errors.New("monster not found")
	}
}

func (r *Registry) RemoveMonster(tenant tenant.Model, uniqueId uint32) (Model, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}

	// First, look up the monster to get its map info (read lock only)
	r.monsterLock.RLock()
	val, ok := r.monsterReg[monKey]
	if !ok {
		r.monsterLock.RUnlock()
		return Model{}, errors.New("monster not found")
	}
	mapKey := NewMapKey(tenant, val.Field())
	r.monsterLock.RUnlock()

	// Acquire locks in same order as CreateMonster: mapLock -> monsterLock
	mapLock := r.getMapLock(mapKey)
	mapLock.Lock()
	defer mapLock.Unlock()

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	// Re-verify the monster still exists (may have been removed concurrently)
	val, ok = r.monsterReg[monKey]
	if !ok {
		return Model{}, errors.New("monster not found")
	}

	if mapMons, ok := r.mapMonsterReg[mapKey]; ok {
		r.mapMonsterReg[mapKey] = removeIfExists(mapMons, val)
	}

	delete(r.monsterReg, monKey)

	// Release the ID back to the allocator for reuse
	r.getOrCreateAllocator(tenant).Release(uniqueId)

	return val, nil
}

func removeIfExists(slice []MonsterKey, value Model) []MonsterKey {
	for i, v := range slice {
		if v.MonsterId == value.UniqueId() {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func (r *Registry) GetMonsters() map[tenant.Model][]Model {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	mons := make(map[tenant.Model][]Model)
	for key, monster := range r.monsterReg {
		var val []Model
		var ok bool
		if val, ok = mons[key.Tenant]; !ok {
			val = make([]Model, 0)
		}
		val = append(val, monster)
		mons[key.Tenant] = val
	}
	return mons
}

func (r *Registry) ApplyStatusEffect(t tenant.Model, uniqueId uint32, effect StatusEffect) (Model, error) {
	monKey := MonsterKey{Tenant: t, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.ApplyStatus(effect)
		r.monsterReg[monKey] = m
		return m, nil
	}
	return Model{}, errors.New("monster not found")
}

func (r *Registry) CancelStatusEffect(t tenant.Model, uniqueId uint32, effectId uuid.UUID) (Model, error) {
	monKey := MonsterKey{Tenant: t, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.CancelStatus(effectId)
		r.monsterReg[monKey] = m
		return m, nil
	}
	return Model{}, errors.New("monster not found")
}

func (r *Registry) CancelStatusEffectByType(t tenant.Model, uniqueId uint32, statusType string) (Model, error) {
	monKey := MonsterKey{Tenant: t, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.CancelStatusByType(statusType)
		r.monsterReg[monKey] = m
		return m, nil
	}
	return Model{}, errors.New("monster not found")
}

func (r *Registry) CancelAllStatusEffects(t tenant.Model, uniqueId uint32) (Model, error) {
	monKey := MonsterKey{Tenant: t, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.CancelAllStatuses()
		r.monsterReg[monKey] = m
		return m, nil
	}
	return Model{}, errors.New("monster not found")
}

func (r *Registry) DeductMp(t tenant.Model, uniqueId uint32, amount uint32) (Model, error) {
	monKey := MonsterKey{Tenant: t, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		m := val.DeductMp(amount)
		r.monsterReg[monKey] = m
		return m, nil
	}
	return Model{}, errors.New("monster not found")
}

func (r *Registry) UpdateStatusEffectLastTick(t tenant.Model, uniqueId uint32, effectId uuid.UUID, tickTime time.Time) (Model, error) {
	monKey := MonsterKey{Tenant: t, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		updated := make([]StatusEffect, 0, len(val.statusEffects))
		for _, se := range val.statusEffects {
			if se.EffectId() == effectId {
				se = se.WithLastTick(tickTime)
			}
			updated = append(updated, se)
		}
		m := Clone(val).Build()
		m.statusEffects = updated
		r.monsterReg[monKey] = m
		return m, nil
	}
	return Model{}, errors.New("monster not found")
}

func (r *Registry) UpdateMonster(t tenant.Model, uniqueId uint32, m Model) {
	monKey := MonsterKey{Tenant: t, MonsterId: uniqueId}

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	r.monsterReg[monKey] = m
}

func (r *Registry) Clear() {
	r.idAllocators = make(map[tenant.Model]*TenantIdAllocator)
	r.mapMonsterReg = make(map[MapKey][]MonsterKey)
	r.mapLocks = make(map[MapKey]*sync.RWMutex)
	r.monsterReg = make(map[MonsterKey]Model)
	r.monsterLock = &sync.RWMutex{}
}
