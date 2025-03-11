package character

import (
	"github.com/Chronicle20/atlas-tenant"
	"sync"
)

type MapKey struct {
	Tenant    tenant.Model
	WorldId   byte
	ChannelId byte
	MapId     uint32
}

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

func (r *Registry) AddCharacter(characterId uint32, mk MapKey) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister[characterId] = mk
}

func (r *Registry) RemoveCharacter(characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.characterRegister, characterId)
}

func (r *Registry) GetLoggedIn() map[uint32]MapKey {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.characterRegister
}
