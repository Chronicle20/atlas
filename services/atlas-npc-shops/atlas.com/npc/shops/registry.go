package shops

import (
	"context"
	"fmt"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg    *atlas.TenantRegistry[uint32, uint32]
	client *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, uint32](client, "npc-shop", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		client: client,
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) shopSetKey(t tenant.Model, shopId uint32) string {
	return fmt.Sprintf("atlas:npc-shop-chars:%s:%d", atlas.TenantKey(t), shopId)
}

func (r *Registry) AddCharacter(ctx context.Context, characterId uint32, templateId uint32) {
	t := tenant.MustFromContext(ctx)

	if oldTemplateId, err := r.reg.Get(ctx, t, characterId); err == nil && oldTemplateId > 0 {
		r.client.SRem(ctx, r.shopSetKey(t, oldTemplateId), strconv.FormatUint(uint64(characterId), 10))
	}

	_ = r.reg.Put(ctx, t, characterId, templateId)

	if templateId > 0 {
		r.client.SAdd(ctx, r.shopSetKey(t, templateId), strconv.FormatUint(uint64(characterId), 10))
	}
}

func (r *Registry) RemoveCharacter(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)

	if templateId, err := r.reg.Get(ctx, t, characterId); err == nil && templateId > 0 {
		r.client.SRem(ctx, r.shopSetKey(t, templateId), strconv.FormatUint(uint64(characterId), 10))
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
	members, err := r.client.SMembers(ctx, r.shopSetKey(t, shopId)).Result()
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
