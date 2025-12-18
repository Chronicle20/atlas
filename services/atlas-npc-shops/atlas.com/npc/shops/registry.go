package shops

import (
	"github.com/google/uuid"
	"sync"
)

type Registry struct {
	mutex             sync.RWMutex
	characterRegister map[uuid.UUID]map[uint32]uint32
	shopCharacterMap  map[uuid.UUID]map[uint32][]uint32
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.characterRegister = make(map[uuid.UUID]map[uint32]uint32)
		registry.shopCharacterMap = make(map[uuid.UUID]map[uint32][]uint32)
	})
	return registry
}

func (r *Registry) ensureTenantMaps(tenantId uuid.UUID) {
	if _, ok := r.characterRegister[tenantId]; !ok {
		r.characterRegister[tenantId] = make(map[uint32]uint32)
	}
	if _, ok := r.shopCharacterMap[tenantId]; !ok {
		r.shopCharacterMap[tenantId] = make(map[uint32][]uint32)
	}
}

func (r *Registry) AddCharacter(tenantId uuid.UUID, characterId uint32, templateId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.ensureTenantMaps(tenantId)

	// If character was already in a shop, remove it from that shop's character list
	if oldTemplateId, ok := r.characterRegister[tenantId][characterId]; ok {
		if oldTemplateId > 0 {
			r.removeFromShopCharacterMap(tenantId, oldTemplateId, characterId)
		}
	}

	// Add character to new shop
	r.characterRegister[tenantId][characterId] = templateId

	// Add character to shop's character list for faster lookups
	if templateId > 0 {
		r.addToShopCharacterMap(tenantId, templateId, characterId)
	}
}

func (r *Registry) removeFromShopCharacterMap(tenantId uuid.UUID, shopId uint32, characterId uint32) {
	if characters, ok := r.shopCharacterMap[tenantId][shopId]; ok {
		for i, id := range characters {
			if id == characterId {
				// Remove character from slice by replacing it with the last element and truncating
				characters[i] = characters[len(characters)-1]
				r.shopCharacterMap[tenantId][shopId] = characters[:len(characters)-1]
				break
			}
		}
		// If shop has no characters, remove the shop entry
		if len(r.shopCharacterMap[tenantId][shopId]) == 0 {
			delete(r.shopCharacterMap[tenantId], shopId)
		}
	}
}

func (r *Registry) addToShopCharacterMap(tenantId uuid.UUID, shopId uint32, characterId uint32) {
	if _, ok := r.shopCharacterMap[tenantId][shopId]; !ok {
		r.shopCharacterMap[tenantId][shopId] = make([]uint32, 0)
	}
	r.shopCharacterMap[tenantId][shopId] = append(r.shopCharacterMap[tenantId][shopId], characterId)
}

func (r *Registry) RemoveCharacter(tenantId uuid.UUID, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.ensureTenantMaps(tenantId)

	// If character was in a shop, remove it from that shop's character list
	if templateId, ok := r.characterRegister[tenantId][characterId]; ok {
		if templateId > 0 {
			r.removeFromShopCharacterMap(tenantId, templateId, characterId)
		}
	}

	// Remove character from register
	delete(r.characterRegister[tenantId], characterId)
}

func (r *Registry) GetShop(tenantId uuid.UUID, characterId uint32) (uint32, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.ensureTenantMaps(tenantId)

	if tid, ok := r.characterRegister[tenantId][characterId]; ok {
		if tid > 0 {
			return tid, true
		}
	}
	return 0, false
}

func (r *Registry) GetCharactersInShop(tenantId uuid.UUID, shopId uint32) []uint32 {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	r.ensureTenantMaps(tenantId)

	// Use the shop-character map for O(1) lookup instead of iterating through all characters
	if characters, ok := r.shopCharacterMap[tenantId][shopId]; ok {
		// Return a copy of the slice to prevent external modifications
		result := make([]uint32, len(characters))
		copy(result, characters)
		return result
	}

	return []uint32{}
}
