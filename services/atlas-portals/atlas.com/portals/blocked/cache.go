package blocked

import (
	"context"
	"fmt"
	"strconv"

	_map "github.com/Chronicle20/atlas-constants/map"
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
		namespace: "blocked-portal",
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) setKey(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("atlas:%s:%s:%s", r.namespace, atlas.TenantKey(t), strconv.FormatUint(uint64(characterId), 10))
}

func portalKey(mapId _map.Id, portalId uint32) string {
	return fmt.Sprintf("%d:%d", mapId, portalId)
}

func parsePortalKey(key string) (_map.Id, uint32) {
	var mapId _map.Id
	var portalId uint32
	fmt.Sscanf(key, "%d:%d", &mapId, &portalId)
	return mapId, portalId
}

func (r *Registry) IsBlocked(ctx context.Context, characterId uint32, mapId _map.Id, portalId uint32) bool {
	t := tenant.MustFromContext(ctx)
	return r.client.SIsMember(ctx, r.setKey(t, characterId), portalKey(mapId, portalId)).Val()
}

func (r *Registry) Block(ctx context.Context, characterId uint32, mapId _map.Id, portalId uint32) {
	t := tenant.MustFromContext(ctx)
	r.client.SAdd(ctx, r.setKey(t, characterId), portalKey(mapId, portalId))
}

func (r *Registry) Unblock(ctx context.Context, characterId uint32, mapId _map.Id, portalId uint32) {
	t := tenant.MustFromContext(ctx)
	r.client.SRem(ctx, r.setKey(t, characterId), portalKey(mapId, portalId))
}

func (r *Registry) ClearForCharacter(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	r.client.Del(ctx, r.setKey(t, characterId))
}

func (r *Registry) GetForCharacter(ctx context.Context, characterId uint32) []Model {
	t := tenant.MustFromContext(ctx)
	members, err := r.client.SMembers(ctx, r.setKey(t, characterId)).Result()
	if err != nil {
		return []Model{}
	}
	result := make([]Model, 0, len(members))
	for _, m := range members {
		mapId, portalId := parsePortalKey(m)
		result = append(result, NewModel(characterId, mapId, portalId))
	}
	return result
}
