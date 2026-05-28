package shops

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg       *atlas.TenantRegistry[uint32, uint32]
	shopChars *atlas.TenantKeyedSet[uint32]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, uint32](client, "npc-shop", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		shopChars: atlas.NewTenantKeyedSet[uint32](client, "npc-shop-chars", func(shopId uint32) string {
			return strconv.FormatUint(uint64(shopId), 10)
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) AddCharacter(ctx context.Context, characterId uint32, templateId uint32) {
	t := tenant.MustFromContext(ctx)

	if oldTemplateId, err := r.reg.Get(ctx, t, characterId); err == nil && oldTemplateId > 0 {
		_ = r.shopChars.Remove(ctx, t, oldTemplateId, strconv.FormatUint(uint64(characterId), 10))
	}

	_ = r.reg.Put(ctx, t, characterId, templateId)

	if templateId > 0 {
		_ = r.shopChars.Add(ctx, t, templateId, strconv.FormatUint(uint64(characterId), 10))
	}
}

func (r *Registry) RemoveCharacter(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)

	if templateId, err := r.reg.Get(ctx, t, characterId); err == nil && templateId > 0 {
		_ = r.shopChars.Remove(ctx, t, templateId, strconv.FormatUint(uint64(characterId), 10))
	}

	_ = r.reg.Remove(ctx, t, characterId)
}

func (r *Registry) GetShop(ctx context.Context, characterId uint32) (uint32, bool) {
	t := tenant.MustFromContext(ctx)
	templateId, err := r.reg.Get(ctx, t, characterId)
	if err != nil || templateId == 0 {
		return 0, false
	}
	return templateId, true
}

func (r *Registry) GetCharactersInShop(ctx context.Context, shopId uint32) []uint32 {
	t := tenant.MustFromContext(ctx)
	members, err := r.shopChars.Members(ctx, t, shopId)
	if err != nil {
		return []uint32{}
	}
	result := make([]uint32, 0, len(members))
	for _, m := range members {
		id, err := strconv.ParseUint(m, 10, 32)
		if err != nil {
			continue
		}
		result = append(result, uint32(id))
	}
	return result
}
