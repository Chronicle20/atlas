package instance

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type CharacterRegistry struct {
	chars *atlas.TenantHash
}

var characterRegistry *CharacterRegistry

func InitCharacterRegistry(client *goredis.Client) {
	characterRegistry = &CharacterRegistry{chars: atlas.NewTenantHash(client, "transport:characters")}
}

func getCharacterRegistry() *CharacterRegistry {
	return characterRegistry
}

// Add registers a character as being in an instance transport.
func (r *CharacterRegistry) Add(ctx context.Context, characterId uint32, instanceId uuid.UUID) {
	t := tenant.MustFromContext(ctx)
	_ = r.chars.Set(ctx, t, strconv.FormatUint(uint64(characterId), 10), instanceId.String())
}

// Remove unregisters a character from instance transport tracking.
func (r *CharacterRegistry) Remove(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.chars.Del(ctx, t, strconv.FormatUint(uint64(characterId), 10))
}

// IsInTransport checks if a character is currently in an instance transport.
func (r *CharacterRegistry) IsInTransport(ctx context.Context, characterId uint32) bool {
	t := tenant.MustFromContext(ctx)
	ok, err := r.chars.Exists(ctx, t, strconv.FormatUint(uint64(characterId), 10))
	if err != nil {
		return false
	}
	return ok
}

// GetInstanceForCharacter returns the instance ID for a character, if any.
func (r *CharacterRegistry) GetInstanceForCharacter(ctx context.Context, characterId uint32) (uuid.UUID, bool) {
	t := tenant.MustFromContext(ctx)
	val, err := r.chars.Get(ctx, t, strconv.FormatUint(uint64(characterId), 10))
	if err != nil {
		return uuid.UUID{}, false
	}
	instanceId, err := uuid.Parse(val)
	if err != nil {
		return uuid.UUID{}, false
	}
	return instanceId, true
}
