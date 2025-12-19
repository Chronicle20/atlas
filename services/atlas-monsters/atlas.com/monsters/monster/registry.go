package monster

import (
	"errors"
	tenant "github.com/Chronicle20/atlas-tenant"
	"sync"
)

type Registry struct {
	mutex  sync.Mutex
	idLock sync.Mutex

	tenantMonsterId map[tenant.Model]uint32
	mapMonsterReg   map[MapKey][]MonsterKey
	mapLocks        map[MapKey]*sync.RWMutex

	monsterReg  map[MonsterKey]Model
	monsterLock *sync.RWMutex
}

var registry *Registry
var once sync.Once

func GetMonsterRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}

		registry.tenantMonsterId = make(map[tenant.Model]uint32)
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

func (r *Registry) CreateMonster(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) Model {
	mapKey := MapKey{Tenant: tenant, WorldId: worldId, ChannelId: channelId, MapId: mapId}

	mapLock := r.getMapLock(mapKey)
	mapLock.Lock()
	defer mapLock.Unlock()

	r.idLock.Lock()
	// TODO need a more efficient mechanism for ID reuse.
	var currentUniqueId = uint32(1000000000)
	if val, ok := r.tenantMonsterId[tenant]; ok {
		currentUniqueId = val
	}

	var ids = make(map[uint32]bool)
	for mk := range r.mapMonsterReg {
		if mk.Tenant == tenant {
			for _, id := range r.mapMonsterReg[mk] {
				ids[id.MonsterId] = true
			}
		}
	}

	for {
		if _, ok := ids[currentUniqueId]; !ok {
			break
		}
		currentUniqueId = currentUniqueId + 1
		if currentUniqueId > 2000000000 {
			currentUniqueId = 1000000000
		}
		r.tenantMonsterId[tenant] = currentUniqueId
	}
	r.idLock.Unlock()

	m := NewMonster(worldId, channelId, mapId, currentUniqueId, monsterId, x, y, fh, stance, team, hp, mp)

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

func (r *Registry) GetMonstersInMap(tenant tenant.Model, worldId byte, channelId byte, mapId uint32) []Model {
	mapKey := NewMapKey(tenant, worldId, channelId, mapId)
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

	r.monsterLock.Lock()
	defer r.monsterLock.Unlock()

	if val, ok := r.monsterReg[monKey]; ok {
		mapKey := NewMapKey(tenant, val.WorldId(), val.ChannelId(), val.MapId())
		mapLock := r.getMapLock(mapKey)
		mapLock.Lock()
		defer mapLock.Unlock()

		if mapMons, ok := r.mapMonsterReg[mapKey]; ok {
			r.mapMonsterReg[mapKey] = removeIfExists(mapMons, val)
		}

		delete(r.monsterReg, monKey)
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

func (r *Registry) Clear() {
	r.tenantMonsterId = make(map[tenant.Model]uint32)
	r.mapMonsterReg = make(map[MapKey][]MonsterKey)
	r.mapLocks = make(map[MapKey]*sync.RWMutex)
	r.monsterReg = make(map[MonsterKey]Model)
	r.monsterLock = &sync.RWMutex{}
}
