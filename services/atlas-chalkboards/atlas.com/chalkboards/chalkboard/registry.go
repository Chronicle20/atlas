package chalkboard

import (
	"sync"

	"github.com/Chronicle20/atlas-tenant"
)

type ChalkboardKey struct {
	Tenant      tenant.Model
	CharacterId uint32
}

type Registry struct {
	mutex             sync.RWMutex
	characterRegister map[ChalkboardKey]string
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterRegister = make(map[ChalkboardKey]string)
	})
	return registry
}

func (r *Registry) Get(t tenant.Model, characterId uint32) (string, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	key := ChalkboardKey{Tenant: t, CharacterId: characterId}
	if val, ok := r.characterRegister[key]; ok {
		return val, ok
	}
	return "", false
}

func (r *Registry) Set(t tenant.Model, characterId uint32, value string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	key := ChalkboardKey{Tenant: t, CharacterId: characterId}
	r.characterRegister[key] = value
}

func (r *Registry) Clear(t tenant.Model, characterId uint32) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	key := ChalkboardKey{Tenant: t, CharacterId: characterId}
	if _, ok := r.characterRegister[key]; ok {
		delete(r.characterRegister, key)
		return true
	}
	return false
}
