package reactor

import (
	"errors"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type registry struct {
	reactors    map[uint32]*Model
	mapReactors map[tenant.Model]map[MapKey][]uint32
	mapLocks    map[tenant.Model]map[MapKey]*sync.Mutex
	tenantLock  map[tenant.Model]*sync.RWMutex
	cooldowns   map[tenant.Model]map[MapKey]map[ReactorKey]time.Time
	lock        sync.RWMutex
}

var once sync.Once
var reg *registry

var runningId = uint32(1000000001)

type MapKey struct {
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	instance  uuid.UUID
}

func NewMapKey(f field.Model) MapKey {
	return MapKey{
		worldId:   f.WorldId(),
		channelId: f.ChannelId(),
		mapId:     f.MapId(),
		instance:  f.Instance(),
	}
}

type ReactorKey struct {
	Classification uint32
	X              int16
	Y              int16
}

func GetRegistry() *registry {
	once.Do(func() {
		reg = &registry{
			reactors:    make(map[uint32]*Model),
			mapReactors: make(map[tenant.Model]map[MapKey][]uint32),
			mapLocks:    make(map[tenant.Model]map[MapKey]*sync.Mutex),
			cooldowns:   make(map[tenant.Model]map[MapKey]map[ReactorKey]time.Time),
			lock:        sync.RWMutex{},
		}
	})
	return reg
}

func (r *registry) Get(id uint32) (Model, error) {
	r.lock.RLock()
	if val, ok := r.reactors[id]; ok {
		r.lock.RUnlock()
		return *val, nil
	} else {
		r.lock.RUnlock()
		return Model{}, errors.New("unable to locate reactor")
	}
}

type Filter func(*Model) bool

func (r *registry) GetAll() map[tenant.Model][]Model {
	r.lock.RLock()
	defer r.lock.RUnlock()

	res := make(map[tenant.Model][]Model)

	for _, m := range r.reactors {
		var val []Model
		var ok bool
		if val, ok = res[m.Tenant()]; !ok {
			val = make([]Model, 0)
		}
		val = append(val, *m)
		res[m.Tenant()] = val
	}
	return res
}

func (r *registry) GetInField(t tenant.Model, f field.Model) []Model {
	mk := NewMapKey(f)

	r.getMapLock(t, mk).Lock()
	defer r.getMapLock(t, mk).Unlock()

	result := make([]Model, 0)

	if _, ok := r.mapReactors[t]; !ok {
		return result
	}

	for _, x := range r.mapReactors[t][mk] {
		result = append(result, *r.reactors[x])
	}

	return result
}

func (r *registry) getMapLock(t tenant.Model, key MapKey) *sync.Mutex {
	var res *sync.Mutex
	r.lock.Lock()
	if _, ok := r.mapLocks[t]; !ok {
		r.mapLocks[t] = make(map[MapKey]*sync.Mutex)
		r.mapReactors[t] = make(map[MapKey][]uint32)
	}
	if _, ok := r.mapLocks[t][key]; !ok {
		r.mapLocks[t][key] = &sync.Mutex{}
		r.mapReactors[t] = make(map[MapKey][]uint32)
	}
	res = r.mapLocks[t][key]
	r.lock.Unlock()
	return res
}

func (r *registry) Create(t tenant.Model, b *ModelBuilder) (Model, error) {
	r.lock.Lock()
	id := r.getNextId()
	m, err := b.SetId(id).UpdateTime().Build()
	if err != nil {
		r.lock.Unlock()
		return Model{}, err
	}
	r.reactors[id] = &m
	r.lock.Unlock()

	mk := NewMapKey(m.Field())
	r.getMapLock(t, mk).Lock()
	defer r.getMapLock(t, mk).Unlock()

	r.mapReactors[t][mk] = append(r.mapReactors[t][mk], m.Id())
	return m, nil
}

func (r *registry) Update(id uint32, modifier func(*ModelBuilder)) (Model, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if val, ok := r.reactors[id]; ok {
		b := NewFromModel(*val)
		modifier(b)
		b.UpdateTime()
		m, err := b.Build()
		if err != nil {
			return Model{}, err
		}
		r.reactors[id] = &m
		return m, nil
	}
	return Model{}, errors.New("unable to locate reactor")
}

func (r *registry) getNextId() uint32 {
	ids := existingIds(r.reactors)

	var currentId = runningId
	for contains(ids, currentId) {
		currentId = currentId + 1
		if currentId > 2000000000 {
			currentId = 1000000001
		}
		runningId = currentId
	}
	return runningId
}

//func (r *registry) Destroy(id uint32) (Model, error) {
//	return r.Update(id, setDestroyed(), updateTime())
//}

func (r *registry) Remove(t tenant.Model, id uint32) {
	r.lock.Lock()
	val, ok := r.reactors[id]
	if !ok {
		return
	}
	delete(r.reactors, id)

	r.lock.Unlock()

	mk := NewMapKey(val.Field())
	r.getMapLock(t, mk).Lock()
	if _, ok := r.mapReactors[t][mk]; ok {
		index := indexOf(id, r.mapReactors[t][mk])
		if index >= 0 && index < len(r.mapReactors[t][mk]) {
			r.mapReactors[t][mk] = remove(r.mapReactors[t][mk], index)
		}
	}
	r.getMapLock(t, mk).Unlock()
}

func existingIds(existing map[uint32]*Model) []uint32 {
	var ids []uint32
	for _, x := range existing {
		ids = append(ids, x.Id())
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

func indexOf(id uint32, data []uint32) int {
	for k, v := range data {
		if id == v {
			return k
		}
	}
	return -1 //not found.
}

func remove(s []uint32, i int) []uint32 {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func (r *registry) RecordCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16, delay uint32) {
	if delay == 0 {
		return
	}

	eligibleAt := time.Now().Add(time.Millisecond * time.Duration(delay))
	rk := ReactorKey{Classification: classification, X: x, Y: y}

	r.getMapLock(t, mk).Lock()
	defer r.getMapLock(t, mk).Unlock()

	r.lock.Lock()
	if _, ok := r.cooldowns[t]; !ok {
		r.cooldowns[t] = make(map[MapKey]map[ReactorKey]time.Time)
	}
	if _, ok := r.cooldowns[t][mk]; !ok {
		r.cooldowns[t][mk] = make(map[ReactorKey]time.Time)
	}
	r.lock.Unlock()

	r.cooldowns[t][mk][rk] = eligibleAt
}

func (r *registry) IsOnCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) bool {
	rk := ReactorKey{Classification: classification, X: x, Y: y}

	r.getMapLock(t, mk).Lock()
	defer r.getMapLock(t, mk).Unlock()

	r.lock.RLock()
	if _, ok := r.cooldowns[t]; !ok {
		r.lock.RUnlock()
		return false
	}
	if _, ok := r.cooldowns[t][mk]; !ok {
		r.lock.RUnlock()
		return false
	}
	r.lock.RUnlock()

	eligibleAt, ok := r.cooldowns[t][mk][rk]
	if !ok {
		return false
	}

	return time.Now().Before(eligibleAt)
}

func (r *registry) ClearCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) {
	rk := ReactorKey{Classification: classification, X: x, Y: y}

	r.getMapLock(t, mk).Lock()
	defer r.getMapLock(t, mk).Unlock()

	r.lock.RLock()
	if _, ok := r.cooldowns[t]; !ok {
		r.lock.RUnlock()
		return
	}
	if _, ok := r.cooldowns[t][mk]; !ok {
		r.lock.RUnlock()
		return
	}
	r.lock.RUnlock()

	delete(r.cooldowns[t][mk], rk)
}

func (r *registry) CleanupExpiredCooldowns() {
	r.lock.Lock()
	defer r.lock.Unlock()

	now := time.Now()
	for t, maps := range r.cooldowns {
		for mk, reactors := range maps {
			for rk, eligibleAt := range reactors {
				if now.After(eligibleAt) {
					delete(r.cooldowns[t][mk], rk)
				}
			}
			if len(r.cooldowns[t][mk]) == 0 {
				delete(r.cooldowns[t], mk)
			}
		}
		if len(r.cooldowns[t]) == 0 {
			delete(r.cooldowns, t)
		}
	}
}
