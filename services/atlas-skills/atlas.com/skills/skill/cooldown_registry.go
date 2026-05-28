package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg     *atlas.TenantRegistry[string, time.Time]
	tenants *atlas.Set
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[string, time.Time](client, "cooldown", func(k string) string {
			return k
		}),
		// "atlas:cooldown:_tenants" — byte-identical to the old hand-rolled key.
		tenants: atlas.NewSet(client, "cooldown:_tenants"),
	}
}

func GetRegistry() *Registry { return registry }

func compositeKey(characterId, skillId uint32) string {
	return fmt.Sprintf("%d:%d", characterId, skillId)
}

func (r *Registry) Apply(ctx context.Context, characterId uint32, skillId uint32, cooldown uint32) error {
	t := tenant.MustFromContext(ctx)
	expiresAt := time.Now().Add(time.Duration(cooldown) * time.Second)
	err := r.reg.Put(ctx, t, compositeKey(characterId, skillId), expiresAt)
	if err != nil {
		return err
	}
	tb, _ := json.Marshal(&t)
	return r.tenants.Add(ctx, string(tb))
}

func (r *Registry) Get(ctx context.Context, characterId uint32, skillId uint32) (time.Time, error) {
	t := tenant.MustFromContext(ctx)
	return r.reg.Get(ctx, t, compositeKey(characterId, skillId))
}

// ClearAll removes all cooldown entries for the given character under the current tenant.
// The prefix "<charId>:" (with trailing colon) is used so that e.g. charId 100
// never accidentally matches charId 1000 or 1001 — a safer invariant than the old
// raw "<charId>*" glob.
func (r *Registry) ClearAll(ctx context.Context, characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	charPrefix := strconv.FormatUint(uint64(characterId), 10) + ":"
	_, err := r.reg.ClearByPrefix(ctx, t, charPrefix)
	return err
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

	members, err := r.tenants.Members(ctx)
	if err != nil {
		return result
	}

	for _, m := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(m), &t); err != nil {
			continue
		}

		entries, err := r.reg.GetAllEntries(ctx, t)
		if err != nil {
			continue
		}

		for suffix, expiresAt := range entries {
			// suffix = "<charId>:<skillId>"
			parts := strings.SplitN(suffix, ":", 2)
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
			result = append(result, CooldownHolder{
				tenant:            t,
				characterId:       uint32(charId),
				skillId:           uint32(sId),
				cooldownExpiresAt: expiresAt,
			})
		}
	}
	return result
}
