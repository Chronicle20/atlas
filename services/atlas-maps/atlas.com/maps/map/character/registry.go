package character

import (
	"sync"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
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

func (r *Registry) GetInMapAllInstances(t tenant.Model, worldId world.Id, channelId channel.Id, mapId _map.Id) []uint32 {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	result := make([]uint32, 0)
	ref := field.NewBuilder(worldId, channelId, mapId).Build()
	for mk, chars := range r.characterRegister {
		if mk.Tenant == t && mk.Field.SameMap(ref) {
			for _, c := range chars {
				result = appendIfMissing(result, c)
			}
		}
	}
	return result
}

func (r *Registry) GetMapsWithCharacters() []MapKey {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]MapKey, 0)
	for mk, mc := range r.characterRegister {
		if len(mc) > 0 {
			result = append(result, mk)
		}
	}
	return result
}
