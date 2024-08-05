package character

import (
	"sync"
)

type Registry struct {
	mutex             sync.Mutex
	characterRegister map[MapKey][]uint32
	mapLocks          map[MapKey]*sync.RWMutex
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}

		registry.characterRegister = make(map[MapKey][]uint32)
		registry.mapLocks = make(map[MapKey]*sync.RWMutex)
	})
	return registry
}

func appendIfMissing(slice []uint32, value uint32) []uint32 {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

func removeIfExists(slice []uint32, value uint32) []uint32 {
	for i, v := range slice {
		if v == value {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func (r *Registry) AddCharacter(key MapKey, characterId uint32) {
	var ml = r.getMapLock(key)
	ml.Lock()
	defer ml.Unlock()

	if _, ok := r.characterRegister[key]; ok {
		r.characterRegister[key] = appendIfMissing(r.characterRegister[key], characterId)
		return
	}
	r.characterRegister[key] = []uint32{characterId}
}

func (r *Registry) getMapLock(key MapKey) *sync.RWMutex {
	var ml *sync.RWMutex
	var ok bool
	if ml, ok = r.mapLocks[key]; !ok {
		r.mutex.Lock()
		r.mapLocks[key] = &sync.RWMutex{}
		ml = r.mapLocks[key]
		r.mutex.Unlock()
	}
	return ml
}

func (r *Registry) RemoveCharacter(key MapKey, characterId uint32) {
	var ml = r.getMapLock(key)
	ml.Lock()
	defer ml.Unlock()

	if _, ok := r.characterRegister[key]; ok {
		r.characterRegister[key] = removeIfExists(r.characterRegister[key], characterId)
	}
}

func (r *Registry) GetInMap(key MapKey) []uint32 {
	ml := r.getMapLock(key)
	ml.RLock()
	defer ml.RUnlock()
	return r.characterRegister[key]
}

func (r *Registry) GetMapsWithCharacters() []MapKey {
	var result = make([]MapKey, 0)
	for mk, ml := range r.mapLocks {
		ml.RLock()
		if mc, ok := r.characterRegister[mk]; ok {
			if len(mc) > 0 {
				result = append(result, mk)
			}
		}
		ml.RUnlock()
	}
	return result
}
