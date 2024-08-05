package monster

import (
	"atlas-monsters/tenant"
	"errors"
	"sync"
)

type Registry struct {
	mutex sync.Mutex

	mapIds        map[MapKey]uint32
	mapMonsterReg map[MapKey][]MonsterKey
	mapLocks      map[MapKey]*sync.RWMutex

	monsterReg   map[MonsterKey]Model
	monsterLocks map[MonsterKey]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetMonsterRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.mapIds = make(map[MapKey]uint32)
		registry.mapMonsterReg = make(map[MapKey][]MonsterKey)
		registry.mapLocks = make(map[MapKey]*sync.RWMutex)

		registry.monsterReg = make(map[MonsterKey]Model)
		registry.monsterLocks = make(map[MonsterKey]*sync.RWMutex)
	})
	return registry
}

func (r *Registry) getMapLock(key MapKey) *sync.RWMutex {
	if val, ok := r.mapLocks[key]; ok {
		return val
	}
	var cm = &sync.RWMutex{}
	r.mutex.Lock()
	r.mapLocks[key] = cm
	r.mapMonsterReg[key] = make([]MonsterKey, 0)
	r.mapIds[key] = uint32(1000000001)
	r.mutex.Unlock()
	return cm
}

func (r *Registry) getMonsterLock(key MonsterKey) *sync.RWMutex {
	if val, ok := r.monsterLocks[key]; ok {
		return val
	}
	var cm = &sync.RWMutex{}
	r.mutex.Lock()
	r.monsterLocks[key] = cm
	r.mutex.Unlock()
	return cm
}

func existingIds(monsters []MonsterKey) []uint32 {
	var ids []uint32
	for _, x := range monsters {
		ids = append(ids, x.MonsterId)
	}
	return ids
}

func contains(ids []uint32, id uint32) bool {
	for _, element := range ids {
		if element == id {
			return true
		}
	}
	return false
}

func (r *Registry) CreateMonster(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) Model {
	mapKey := MapKey{Tenant: tenant, WorldId: worldId, ChannelId: channelId, MapId: mapId}

	mapLock := r.getMapLock(mapKey)
	mapLock.Lock()
	defer mapLock.Unlock()

	var existingIds = existingIds(r.mapMonsterReg[mapKey])

	var currentUniqueId = r.mapIds[mapKey]
	for contains(existingIds, currentUniqueId) {
		currentUniqueId = currentUniqueId + 1
		if currentUniqueId > 2000000000 {
			currentUniqueId = 1000000001
		}
		r.mapIds[mapKey] = currentUniqueId
	}

	m := NewMonster(worldId, channelId, mapId, currentUniqueId, monsterId, x, y, fh, stance, team, hp, mp)

	monKey := MonsterKey{Tenant: tenant, MonsterId: m.UniqueId()}
	r.mapMonsterReg[mapKey] = append(r.mapMonsterReg[mapKey], monKey)

	monLock := r.getMonsterLock(monKey)
	monLock.Lock()
	defer monLock.Unlock()

	r.monsterReg[monKey] = m
	return m
}

func (r *Registry) GetMonster(tenant tenant.Model, uniqueId uint32) (Model, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}
	monLock := r.getMonsterLock(monKey)
	monLock.RLock()
	defer monLock.RUnlock()

	if m, ok := r.monsterReg[monKey]; ok {
		return m, nil
	}
	return Model{}, errors.New("monster not found")
}

func (r *Registry) GetMonstersInMap(tenant tenant.Model, worldId byte, channelId byte, mapId uint32) []Model {
	mapKey := NewMapKey(tenant, worldId, channelId, mapId)
	mapLock := r.getMapLock(mapKey)
	mapLock.RLock()
	defer mapLock.RUnlock()

	var result []Model
	for _, monKey := range r.mapMonsterReg[mapKey] {
		monLock := r.getMonsterLock(monKey)
		monLock.RLock()
		if m, ok := r.monsterReg[monKey]; ok {
			result = append(result, m)
		}
		monLock.RUnlock()
	}
	return result
}

func (r *Registry) MoveMonster(tenant tenant.Model, uniqueId uint32, endX int16, endY int16, stance byte) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}
	monLock := r.getMonsterLock(monKey)
	monLock.Lock()
	defer monLock.Unlock()
	if val, ok := r.monsterReg[monKey]; ok {
		r.monsterReg[monKey] = val.Move(endX, endY, stance)
	}
}

func (r *Registry) ControlMonster(tenant tenant.Model, uniqueId uint32, characterId uint32) (Model, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}
	monLock := r.getMonsterLock(monKey)
	monLock.Lock()
	defer monLock.Unlock()
	if val, ok := r.monsterReg[monKey]; ok {
		r.monsterReg[monKey] = val.Control(characterId)
		return r.monsterReg[monKey], nil
	} else {
		return Model{}, errors.New("monster not found")
	}
}

func (r *Registry) ClearControl(tenant tenant.Model, uniqueId uint32) (Model, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}
	monLock := r.getMonsterLock(monKey)
	monLock.Lock()
	defer monLock.Unlock()
	if val, ok := r.monsterReg[monKey]; ok {
		r.monsterReg[monKey] = val.ClearControl()
		return r.monsterReg[monKey], nil
	} else {
		return Model{}, errors.New("monster not found")
	}
}

func (r *Registry) ApplyDamage(tenant tenant.Model, characterId uint32, damage int64, uniqueId uint32) (DamageSummary, error) {
	monKey := MonsterKey{Tenant: tenant, MonsterId: uniqueId}
	monLock := r.getMonsterLock(monKey)
	monLock.Lock()
	defer monLock.Unlock()
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
	monLock := r.getMonsterLock(monKey)
	monLock.Lock()
	defer monLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		mapKey := NewMapKey(tenant, val.WorldId(), val.ChannelId(), val.MapId())
		mapLock := r.getMapLock(mapKey)
		mapLock.Lock()
		defer mapLock.Unlock()

		if mapMons, ok := r.mapMonsterReg[mapKey]; ok {
			r.mapMonsterReg[mapKey] = removeIfExists(mapMons, val)
		}

		delete(r.monsterReg, monKey)
		delete(r.monsterLocks, monKey)
		return val, nil
	}
	return Model{}, errors.New("monster not found")
}

func removeIfExists(slice []MonsterKey, value Model) []MonsterKey {
	for i, v := range slice {
		if v.MonsterId == value.UniqueId() {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func (r *Registry) GetMonsters() []Model {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	mons := make([]Model, 0)
	for monKey, monLock := range r.monsterLocks {
		monLock.RLock()
		if m, ok := r.monsterReg[monKey]; ok {
			mons = append(mons, m)
		}
		monLock.RUnlock()
	}
	return mons
}
