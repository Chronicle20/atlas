package character

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-constants/field"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type MapKey struct {
	Tenant tenant.Model
	Field  field.Model
}

type Registry struct {
	reg    *atlas.TenantRegistry[uint32, field.Model]
	client *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, field.Model](client, "pet-character", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		client: client,
	}
}

func GetRegistry() *Registry { return registry }

func (r *Registry) tenantSetKey() string {
	return fmt.Sprintf("atlas:%s:_tenants", r.reg.Namespace())
}

func (r *Registry) AddCharacter(ctx context.Context, characterId uint32, f field.Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Put(ctx, t, characterId, f)
	tb, _ := json.Marshal(&t)
	r.client.SAdd(ctx, r.tenantSetKey(), tb)
}

func (r *Registry) RemoveCharacter(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Remove(ctx, t, characterId)
}

func (r *Registry) GetLoggedIn(ctx context.Context) (map[uint32]MapKey, error) {
	result := make(map[uint32]MapKey)

	members, err := r.client.SMembers(ctx, r.tenantSetKey()).Result()
	if err != nil {
		return result, err
	}

	for _, m := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(m), &t); err != nil {
			continue
		}

		pattern := fmt.Sprintf("atlas:%s:%s:*", r.reg.Namespace(), atlas.TenantKey(t))
		prefix := fmt.Sprintf("atlas:%s:%s:", r.reg.Namespace(), atlas.TenantKey(t))
		var cursor uint64
		for {
			keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}
			if len(keys) > 0 {
				pipe := r.client.Pipeline()
				cmds := make([]*goredis.StringCmd, len(keys))
				for i, k := range keys {
					cmds[i] = pipe.Get(ctx, k)
				}
				_, _ = pipe.Exec(ctx)

				for i, cmd := range cmds {
					data, err := cmd.Bytes()
					if err != nil {
						continue
					}
					keySuffix := strings.TrimPrefix(keys[i], prefix)
					if strings.HasPrefix(keySuffix, "_") {
						continue
					}
					charId, err := strconv.ParseUint(keySuffix, 10, 32)
					if err != nil {
						continue
					}
					var f field.Model
					if err := json.Unmarshal(data, &f); err != nil {
						continue
					}
					result[uint32(charId)] = MapKey{Tenant: t, Field: f}
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
	return result, nil
}
