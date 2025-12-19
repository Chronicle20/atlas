package reactor

import (
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type registry struct {
	reactors    map[uint32]*Model
	mapReactors map[tenant.Model]map[MapKey][]uint32
	mapLocks    map[tenant.Model]map[MapKey]*sync.Mutex
	tenantLock  map[tenant.Model]*sync.RWMutex
	lock        sync.RWMutex
}

var once sync.Once
var reg *registry

var runningId = uint32(1000000001)

type MapKey struct {
	worldId   byte
	channelId byte
	mapId     uint32
}

func GetRegistry() *registry {
	once.Do(func() {
		reg = &registry{
			reactors:    make(map[uint32]*Model),
			mapReactors: make(map[tenant.Model]map[MapKey][]uint32),
			mapLocks:    make(map[tenant.Model]map[MapKey]*sync.Mutex),
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

func (r *registry) GetInMap(t tenant.Model, worldId byte, channelId byte, mapId uint32) []Model {
	mk := MapKey{worldId, channelId, mapId}

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

func (r *registry) Create(t tenant.Model, b *ModelBuilder) Model {
	r.lock.Lock()
	id := r.getNextId()
	m := b.SetId(id).UpdateTime().Build()
	r.reactors[id] = &m
	r.lock.Unlock()

	mk := MapKey{m.WorldId(), m.ChannelId(), m.MapId()}
	r.getMapLock(t, mk).Lock()
	defer r.getMapLock(t, mk).Unlock()

	r.mapReactors[t][mk] = append(r.mapReactors[t][mk], m.Id())
	return m
}

//func (r *registry) Update(id uint32, modifiers ...Modifier) (Model, error) {
//	r.lock.Lock()
//	if val, ok := r.reactors[id]; ok {
//		r.lock.Unlock()
//		for _, modifier := range modifiers {
//			modifier(val)
//		}
//		updateTime()(val)
//		r.reactors[id] = val
//		return *val, nil
//	} else {
//		r.lock.Unlock()
//		return Model{}, errors.New("unable to locate reactor")
//	}
//}

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

	mk := MapKey{val.WorldId(), val.ChannelId(), val.MapId()}
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
