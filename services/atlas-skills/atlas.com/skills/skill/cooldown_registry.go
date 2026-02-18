package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg    *atlas.TenantRegistry[string, time.Time]
	client *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[string, time.Time](client, "cooldown", func(k string) string {
			return k
		}),
		client: client,
	}
}

func GetRegistry() *Registry { return registry }

func compositeKey(characterId, skillId uint32) string {
	return fmt.Sprintf("%d:%d", characterId, skillId)
}

func (r *Registry) tenantSetKey() string {
	return fmt.Sprintf("atlas:%s:_tenants", r.reg.Namespace())
}

func (r *Registry) Apply(ctx context.Context, characterId uint32, skillId uint32, cooldown uint32) error {
	t := tenant.MustFromContext(ctx)
	expiresAt := time.Now().Add(time.Duration(cooldown) * time.Second)
	err := r.reg.Put(ctx, t, compositeKey(characterId, skillId), expiresAt)
	if err != nil {
		return err
	}
	tb, _ := json.Marshal(&t)
	r.client.SAdd(ctx, r.tenantSetKey(), tb)
	return nil
}

func (r *Registry) Get(ctx context.Context, characterId uint32, skillId uint32) (time.Time, error) {
	t := tenant.MustFromContext(ctx)
	return r.reg.Get(ctx, t, compositeKey(characterId, skillId))
}

func (r *Registry) ClearAll(ctx context.Context, characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	charPrefix := strconv.FormatUint(uint64(characterId), 10) + ":"
	pattern := fmt.Sprintf("atlas:%s:%s:%s*", r.reg.Namespace(), atlas.TenantKey(t), charPrefix)

	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			r.client.Del(ctx, keys...)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (r *Registry) Clear(ctx context.Context, characterId uint32, skillId uint32) error {
	t := tenant.MustFromContext(ctx)
	return r.reg.Remove(ctx, t, compositeKey(characterId, skillId))
}

type CooldownHolder struct {
	tenant            tenant.Model
	characterId       uint32
	skillId           uint32
	cooldownExpiresAt time.Time
}

func (h CooldownHolder) CooldownExpiresAt() time.Time {
	return h.cooldownExpiresAt
}

func (h CooldownHolder) Tenant() tenant.Model {
	return h.tenant
}

func (h CooldownHolder) CharacterId() uint32 {
	return h.characterId
}

func (h CooldownHolder) SkillId() uint32 {
	return h.skillId
}

func (r *Registry) GetAll(ctx context.Context) []CooldownHolder {
	result := make([]CooldownHolder, 0)

	members, err := r.client.SMembers(ctx, r.tenantSetKey()).Result()
	if err != nil {
		return result
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
					parts := strings.SplitN(keySuffix, ":", 2)
					if len(parts) != 2 {
						continue
					}
					charId, err := strconv.ParseUint(parts[0], 10, 32)
					if err != nil {
						continue
					}
					sId, err := strconv.ParseUint(parts[1], 10, 32)
					if err != nil {
						continue
					}
					var expiresAt time.Time
					if err := json.Unmarshal(data, &expiresAt); err != nil {
						continue
					}
					result = append(result, CooldownHolder{
						tenant:            t,
						characterId:       uint32(charId),
						skillId:           uint32(sId),
						cooldownExpiresAt: expiresAt,
					})
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
	return result
}
