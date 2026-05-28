package character

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type MapKey struct {
	Tenant tenant.Model
	Field  field.Model
}

type Registry struct {
	reg     *atlas.TenantRegistry[uint32, field.Model]
	tenants *atlas.Set
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, field.Model](client, "pet-character", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		tenants: atlas.NewSet(client, "pet-character:_tenants"),
	}
}

func GetRegistry() *Registry { return registry }

func (r *Registry) AddCharacter(ctx context.Context, characterId uint32, f field.Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Put(ctx, t, characterId, f)
	if tb, err := json.Marshal(&t); err == nil {
		_ = r.tenants.Add(ctx, string(tb))
	}
}

func (r *Registry) RemoveCharacter(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Remove(ctx, t, characterId)
}

func (r *Registry) GetLoggedIn(ctx context.Context) (map[uint32]MapKey, error) {
	result := make(map[uint32]MapKey)

	members, err := r.tenants.Members(ctx)
	if err != nil {
		return result, err
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

		for charIdStr, f := range entries {
			charId, err := strconv.ParseUint(charIdStr, 10, 32)
			if err != nil {
				continue
			}
			result[uint32(charId)] = MapKey{Tenant: t, Field: f}
		}
	}
	return result, nil
}
