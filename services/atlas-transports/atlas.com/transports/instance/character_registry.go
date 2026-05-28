package instance

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type CharacterRegistry struct {
	chars *atlas.Hash
}

var characterRegistry *CharacterRegistry

func InitCharacterRegistry(client *goredis.Client) {
	characterRegistry = &CharacterRegistry{chars: atlas.NewHash(client, "transport:characters")}
}

func getCharacterRegistry() *CharacterRegistry {
	return characterRegistry
}

// Add registers a character as being in an instance transport.
func (r *CharacterRegistry) Add(characterId uint32, instanceId uuid.UUID) {
	_ = r.chars.Set(context.Background(), strconv.FormatUint(uint64(characterId), 10), instanceId.String())
}

// Remove unregisters a character from instance transport tracking.
func (r *CharacterRegistry) Remove(characterId uint32) {
	_ = r.chars.Del(context.Background(), strconv.FormatUint(uint64(characterId), 10))
}

// IsInTransport checks if a character is currently in an instance transport.
func (r *CharacterRegistry) IsInTransport(characterId uint32) bool {
	ok, err := r.chars.Exists(context.Background(), strconv.FormatUint(uint64(characterId), 10))
	if err != nil {
		return false
	}
	return ok
}

// GetInstanceForCharacter returns the instance ID for a character, if any.
func (r *CharacterRegistry) GetInstanceForCharacter(characterId uint32) (uuid.UUID, bool) {
	val, err := r.chars.Get(context.Background(), strconv.FormatUint(uint64(characterId), 10))
	if err != nil {
		return uuid.UUID{}, false
	}
	instanceId, err := uuid.Parse(val)
	if err != nil {
		return uuid.UUID{}, false
	}
	return instanceId, true
}
