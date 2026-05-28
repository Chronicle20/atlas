package character

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	sets   *atlas.TenantKeyedSet[field.Model]
	client *goredis.Client // retained for ResetForTesting only
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		sets: atlas.NewTenantKeyedSet[field.Model](client, "chair-char", func(f field.Model) string {
			return fmt.Sprintf("%d:%d:%d:%s", f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
		}),
		client: client,
	}
}

func getRegistry() *Registry {
	return registry
}

func (r *Registry) AddCharacter(ctx context.Context, key MapKey, characterId uint32) {
	_ = r.sets.Add(ctx, key.Tenant, key.Field, strconv.FormatUint(uint64(characterId), 10))
}

func (r *Registry) RemoveCharacter(ctx context.Context, key MapKey, characterId uint32) {
	_ = r.sets.Remove(ctx, key.Tenant, key.Field, strconv.FormatUint(uint64(characterId), 10))
}

func (r *Registry) GetInMap(ctx context.Context, key MapKey) []uint32 {
	members, err := r.sets.Members(ctx, key.Tenant, key.Field)
	if err != nil {
		return nil
	}
	result := make([]uint32, 0, len(members))
	for _, m := range members {
		v, err := strconv.ParseUint(m, 10, 32)
		if err != nil {
			continue
		}
		result = append(result, uint32(v))
	}
	return result
}

// ResetForTesting clears all registry state. Only for use in tests.
// FlushDB: test-only full reset; tests use an isolated miniredis instance.
func (r *Registry) ResetForTesting(ctx context.Context, t tenant.Model) {
	_ = r.client.FlushDB(ctx).Err()
}
