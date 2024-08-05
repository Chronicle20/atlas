package monster

import (
	"errors"
	"github.com/google/uuid"
	"sync"
)

type Registry struct {
	mutex                 sync.Mutex
	monsterRegisterRWLock sync.RWMutex
	monsterRegister       map[uuid.UUID]map[uint32]Model
	mapMonsters           map[MapKey][]uint32
	mapLocks              map[MapKey]*sync.Mutex
}

var monsterRegistry *Registry
var once sync.Once

var uniqueId = uint32(1000000001)

func GetMonsterRegistry() *Registry {
	once.Do(func() {
		monsterRegistry = &Registry{}

		monsterRegistry.monsterRegister = make(map[uuid.UUID]map[uint32]Model)
		monsterRegistry.mapMonsters = make(map[MapKey][]uint32)

		monsterRegistry.mapLocks = make(map[MapKey]*sync.Mutex)
	})
	return monsterRegistry
}

func (r *Registry) getMapLock(key MapKey) *sync.Mutex {
	if val, ok := r.mapLocks[key]; ok {
		return val
	} else {
		var cm = &sync.Mutex{}
		r.mutex.Lock()
		r.mapLocks[key] = cm
		r.mutex.Unlock()
		return cm
	}
}

func existingIds(monsters map[uint32]Model) []uint32 {
	var ids []uint32
	for _, x := range monsters {
		ids = append(ids, x.UniqueId())
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

func (r *Registry) CreateMonster(tenantId uuid.UUID, worldId byte, channelId byte, mapId uint32, monsterId uint32, x int, y int, fh int, stance int, team int, hp uint32, mp uint32) Model {
	r.monsterRegisterRWLock.Lock()
	var existingIds = existingIds(r.monsterRegister[tenantId])

	var currentUniqueId = uniqueId
	for contains(existingIds, currentUniqueId) {
		currentUniqueId = currentUniqueId + 1
		if currentUniqueId > 2000000000 {
			currentUniqueId = 1000000001
		}
		uniqueId = currentUniqueId
	}

	m := NewMonster(worldId, channelId, mapId, currentUniqueId, monsterId, x, y, fh, stance, team, hp, mp)

	r.monsterRegister[tenantId][uniqueId] = m
	r.monsterRegisterRWLock.Unlock()

	mk := NewMapKey(tenantId, worldId, channelId, mapId)
	r.getMapLock(mk).Lock()
	if om, ok := r.mapMonsters[mk]; ok {
		r.mapMonsters[mk] = append(om, m.UniqueId())
	} else {
		r.mapMonsters[mk] = append([]uint32{}, m.UniqueId())
	}
	r.getMapLock(mk).Unlock()

	return m
}

func (r *Registry) GetMonster(tenantId uuid.UUID, uniqueId uint32) (Model, error) {
	r.monsterRegisterRWLock.RLock()
	if val, ok := r.monsterRegister[tenantId][uniqueId]; ok {
		r.monsterRegisterRWLock.RUnlock()
		return val, nil
	} else {
		r.monsterRegisterRWLock.RUnlock()
		return Model{}, errors.New("monster not found")
	}
}

func (r *Registry) GetMonstersInMap(tenantId uuid.UUID, worldId byte, channelId byte, mapId uint32) []Model {
	mk := NewMapKey(tenantId, worldId, channelId, mapId)
	r.getMapLock(mk).Lock()
	r.monsterRegisterRWLock.RLock()
	var result []Model
	for _, x := range r.mapMonsters[mk] {
		result = append(result, r.monsterRegister[tenantId][x])
	}
	r.monsterRegisterRWLock.RUnlock()
	r.getMapLock(mk).Unlock()
	return result
}

func (r *Registry) MoveMonster(tenantId uuid.UUID, uniqueId uint32, endX int, endY int, stance int) {
	r.monsterRegisterRWLock.Lock()
	if m, ok := r.monsterRegister[tenantId][uniqueId]; ok {
		um := m.Move(endX, endY, stance)
		r.monsterRegister[tenantId][uniqueId] = um
	}
	r.monsterRegisterRWLock.Unlock()
}

func (r *Registry) ControlMonster(tenantId uuid.UUID, uniqueId uint32, characterId uint32) (Model, error) {
	r.monsterRegisterRWLock.Lock()
	if m, ok := r.monsterRegister[tenantId][uniqueId]; ok {
		um := m.Control(characterId)
		r.monsterRegister[tenantId][uniqueId] = um
		r.monsterRegisterRWLock.Unlock()
		return um, nil
	} else {
		r.monsterRegisterRWLock.Unlock()
		return m, errors.New("monster not found")
	}
}

func (r *Registry) ClearControl(tenantId uuid.UUID, uniqueId uint32) (Model, error) {
	r.monsterRegisterRWLock.Lock()
	if m, ok := r.monsterRegister[tenantId][uniqueId]; ok {
		um := m.ClearControl()
		r.monsterRegister[tenantId][uniqueId] = um
		r.monsterRegisterRWLock.Unlock()
		return um, nil
	} else {
		r.monsterRegisterRWLock.Unlock()
		return m, errors.New("monster not found")
	}
}

func (r *Registry) ApplyDamage(tenantId uuid.UUID, characterId uint32, damage int64, uniqueId uint32) (*DamageSummary, error) {
	r.monsterRegisterRWLock.Lock()
	if m, ok := r.monsterRegister[tenantId][uniqueId]; ok {
		um := m.Damage(characterId, damage)
		r.monsterRegister[tenantId][uniqueId] = um
		r.monsterRegisterRWLock.Unlock()
		return &DamageSummary{
			CharacterId:   characterId,
			Monster:       um,
			VisibleDamage: damage,
			ActualDamage:  int64(m.Hp() - um.Hp()),
			Killed:        um.Hp() == 0,
		}, nil
	} else {
		r.monsterRegisterRWLock.Unlock()
		return nil, errors.New("monster not found")
	}
}

func (r *Registry) RemoveMonster(tenantId uuid.UUID, uniqueId uint32) (Model, error) {
	r.monsterRegisterRWLock.Lock()
	if m, ok := r.monsterRegister[tenantId][uniqueId]; ok {
		mk := NewMapKey(tenantId, m.WorldId(), m.ChannelId(), m.MapId())
		r.removeMonster(mk, uniqueId)
		return m, nil
	}
	r.monsterRegisterRWLock.Unlock()
	return Model{}, errors.New("monster not found")
}

func remove(c []uint32, i int) []uint32 {
	c[i] = c[len(c)-1]
	return c[:len(c)-1]
}

func indexOf(id uint32, data []uint32) int {
	for k, v := range data {
		if id == v {
			return k
		}
	}
	return -1 //not found.
}

func (r *Registry) removeMonster(mapId MapKey, uniqueId uint32) {
	index := indexOf(uniqueId, r.mapMonsters[mapId])
	if index >= 0 && index < len(r.mapMonsters[mapId]) {
		r.mapMonsters[mapId] = remove(r.mapMonsters[mapId], index)
	}
}

func (r *Registry) GetMonsters(tenantId uuid.UUID) []Model {
	r.monsterRegisterRWLock.RLock()
	ms := make([]Model, len(r.monsterRegister[tenantId]))
	for _, x := range r.monsterRegister[tenantId] {
		ms = append(ms, x)
	}
	r.monsterRegisterRWLock.RUnlock()
	return ms
}
