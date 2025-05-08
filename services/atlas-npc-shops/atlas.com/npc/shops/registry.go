package shops

import (
	"sync"
)

type Registry struct {
	mutex             sync.RWMutex
	characterRegister map[uint32]uint32
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}

		registry.characterRegister = make(map[uint32]uint32)
	})
	return registry
}

func (r *Registry) AddCharacter(characterId uint32, templateId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister[characterId] = templateId
}

func (r *Registry) RemoveCharacter(characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.characterRegister, characterId)
}

func (r *Registry) GetShop(characterId uint32) (uint32, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if tid, ok := r.characterRegister[characterId]; ok {
		if tid > 0 {
			return tid, true
		}
	}
	return 0, false
}
