package character

import (
	"sync"
)

type Registry struct {
	mutex             sync.RWMutex
	characterRegister map[MapKey][]uint32
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterRegister = make(map[MapKey][]uint32)
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
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.characterRegister[key]; !ok {
		r.characterRegister[key] = make([]uint32, 0)
	}
	r.characterRegister[key] = appendIfMissing(r.characterRegister[key], characterId)
}

func (r *Registry) RemoveCharacter(key MapKey, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.characterRegister[key]; ok {
		r.characterRegister[key] = removeIfExists(r.characterRegister[key], characterId)
	}
}

func (r *Registry) GetInMap(key MapKey) []uint32 {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.characterRegister[key]
}
