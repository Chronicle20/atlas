package instance

import (
	"sync"

	"github.com/google/uuid"
)

type CharacterRegistry struct {
	mu          sync.RWMutex
	byCharacter map[uint32]uuid.UUID
}

var characterRegistry *CharacterRegistry
var characterRegistryOnce sync.Once

func getCharacterRegistry() *CharacterRegistry {
	characterRegistryOnce.Do(func() {
		characterRegistry = &CharacterRegistry{
			byCharacter: make(map[uint32]uuid.UUID),
		}
	})
	return characterRegistry
}

// Add registers a character as being in an instance transport.
func (r *CharacterRegistry) Add(characterId uint32, instanceId uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byCharacter[characterId] = instanceId
}

// Remove unregisters a character from instance transport tracking.
func (r *CharacterRegistry) Remove(characterId uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byCharacter, characterId)
}

// IsInTransport checks if a character is currently in an instance transport.
func (r *CharacterRegistry) IsInTransport(characterId uint32) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.byCharacter[characterId]
	return ok
}

// GetInstanceForCharacter returns the instance ID for a character, if any.
func (r *CharacterRegistry) GetInstanceForCharacter(characterId uint32) (uuid.UUID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	instanceId, ok := r.byCharacter[characterId]
	return instanceId, ok
}
