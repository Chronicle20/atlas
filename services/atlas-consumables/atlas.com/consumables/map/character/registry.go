package character

import (
	"sync"
)

type Registry struct {
	mutex             sync.RWMutex
	characterRegister map[uint32]MapKey
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterRegister = make(map[uint32]MapKey)
	})
	return registry
}

func (r *Registry) AddCharacter(key MapKey, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister[characterId] = key
}

func (r *Registry) RemoveCharacter(characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.characterRegister, characterId)
}

func (r *Registry) GetMap(characterId uint32) (MapKey, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	mk, ok := r.characterRegister[characterId]
	return mk, ok
}
