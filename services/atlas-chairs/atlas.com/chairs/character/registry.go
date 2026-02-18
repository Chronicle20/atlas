package character

import (
	"context"
	"fmt"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	client    *goredis.Client
	namespace string
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		client:    client,
		namespace: "chair-char",
	}
}

func getRegistry() *Registry {
	return registry
}

func (r *Registry) setKey(key MapKey) string {
	tk := atlas.TenantKey(key.Tenant)
	f := key.Field
	return fmt.Sprintf("atlas:%s:%s:%d:%d:%d:%s",
		r.namespace, tk,
		f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

func (r *Registry) AddCharacter(ctx context.Context, key MapKey, characterId uint32) {
	rk := r.setKey(key)
	r.client.SAdd(ctx, rk, strconv.FormatUint(uint64(characterId), 10))
}

func (r *Registry) RemoveCharacter(ctx context.Context, key MapKey, characterId uint32) {
	rk := r.setKey(key)
	r.client.SRem(ctx, rk, strconv.FormatUint(uint64(characterId), 10))
}

func (r *Registry) GetInMap(ctx context.Context, key MapKey) []uint32 {
	rk := r.setKey(key)
	members, err := r.client.SMembers(ctx, rk).Result()
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
func (r *Registry) ResetForTesting(ctx context.Context, t tenant.Model) {
	// Flush by scanning and deleting keys with our namespace prefix.
	pattern := fmt.Sprintf("atlas:%s:*", r.namespace)
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			r.client.Del(ctx, keys...)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}
