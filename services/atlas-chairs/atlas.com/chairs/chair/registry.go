package chair

import (
	"sync"
)

type Registry struct {
	mutex             sync.RWMutex
	characterRegister map[uint32]Model
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterRegister = make(map[uint32]Model)
	})
	return registry
}

func (r *Registry) Get(characterId uint32) (Model, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if val, ok := r.characterRegister[characterId]; ok {
		return val, ok
	}
	return Model{}, false
}

func (r *Registry) Set(characterId uint32, value Model) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister[characterId] = value
}

func (r *Registry) Clear(characterId uint32) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.characterRegister[characterId]; ok {
		delete(r.characterRegister, characterId)
		return true
	}
	return false
}
