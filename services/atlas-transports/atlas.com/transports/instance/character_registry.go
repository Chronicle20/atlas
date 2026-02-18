package instance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const characterHashKey = "transport:characters"

type CharacterRegistry struct {
	client *goredis.Client
}

var characterRegistry *CharacterRegistry

func InitCharacterRegistry(client *goredis.Client) {
	characterRegistry = &CharacterRegistry{client: client}
}

func getCharacterRegistry() *CharacterRegistry {
	return characterRegistry
}

// Add registers a character as being in an instance transport.
func (r *CharacterRegistry) Add(characterId uint32, instanceId uuid.UUID) {
	_ = r.client.HSet(context.Background(), characterHashKey, fmt.Sprintf("%d", characterId), instanceId.String()).Err()
}

// Remove unregisters a character from instance transport tracking.
func (r *CharacterRegistry) Remove(characterId uint32) {
	_ = r.client.HDel(context.Background(), characterHashKey, fmt.Sprintf("%d", characterId)).Err()
}

// IsInTransport checks if a character is currently in an instance transport.
func (r *CharacterRegistry) IsInTransport(characterId uint32) bool {
	exists, err := r.client.HExists(context.Background(), characterHashKey, fmt.Sprintf("%d", characterId)).Result()
	if err != nil {
		return false
	}
	return exists
}

// GetInstanceForCharacter returns the instance ID for a character, if any.
func (r *CharacterRegistry) GetInstanceForCharacter(characterId uint32) (uuid.UUID, bool) {
	val, err := r.client.HGet(context.Background(), characterHashKey, fmt.Sprintf("%d", characterId)).Result()
	if err != nil {
		return uuid.UUID{}, false
	}
	instanceId, err := uuid.Parse(val)
	if err != nil {
		return uuid.UUID{}, false
	}
	return instanceId, true
}
