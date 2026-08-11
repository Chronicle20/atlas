package character

import (
	"context"
	"fmt"
	"strconv"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
)

type Registry struct {
	sets *atlas.TenantKeyedSet[field.Model]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		sets: atlas.NewTenantKeyedSet[field.Model](client, "kite-char", func(f field.Model) string {
			return fmt.Sprintf("%d:%d:%d:%s", f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
		}),
	}
}

func getRegistry() *Registry {
	return registry
}

func (r *Registry) AddCharacter(ctx context.Context, key MapKey, characterId uint32) error {
	return r.sets.Add(ctx, key.Tenant, key.Field, strconv.FormatUint(uint64(characterId), 10))
}

func (r *Registry) RemoveCharacter(ctx context.Context, key MapKey, characterId uint32) error {
	return r.sets.Remove(ctx, key.Tenant, key.Field, strconv.FormatUint(uint64(characterId), 10))
}

// GetInMap returns every characterId currently indexed under key. A Redis
// failure is propagated rather than coerced to an empty slice -- callers
// (kite.ProcessorImpl.InMapModelProvider in particular) rely on this to fail
// the per-map MaxPerMap cap check loudly instead of under-counting on a
// Redis blip.
func (r *Registry) GetInMap(ctx context.Context, key MapKey) ([]uint32, error) {
	members, err := r.sets.Members(ctx, key.Tenant, key.Field)
	if err != nil {
		return nil, err
	}
	result := make([]uint32, 0, len(members))
	for _, m := range members {
		v, err := strconv.ParseUint(m, 10, 32)
		if err != nil {
			continue
		}
		result = append(result, uint32(v))
	}
	return result, nil
}
