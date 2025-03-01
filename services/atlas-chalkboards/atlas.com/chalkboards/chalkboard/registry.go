package chalkboard

import "sync"

type Registry struct {
	mutex             sync.RWMutex
	characterRegister map[uint32]string
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterRegister = make(map[uint32]string)
	})
	return registry
}

func (r *Registry) Get(characterId uint32) (string, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if val, ok := r.characterRegister[characterId]; ok {
		return val, ok
	}
	return "", false
}

func (r *Registry) Set(characterId uint32, value string) {
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
