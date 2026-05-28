package blocked

import (
	"context"
	"fmt"
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	sets *atlas.TenantKeyedSet[uint32]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		sets: atlas.NewTenantKeyedSet[uint32](client, "blocked-portal", func(characterId uint32) string {
			return strconv.FormatUint(uint64(characterId), 10)
		}),
	}
}

func GetRegistry() *Registry {
	return registry
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
	ok, _ := r.sets.IsMember(ctx, t, characterId, portalKey(mapId, portalId))
	return ok
}

func (r *Registry) Block(ctx context.Context, characterId uint32, mapId _map.Id, portalId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.sets.Add(ctx, t, characterId, portalKey(mapId, portalId))
}

func (r *Registry) Unblock(ctx context.Context, characterId uint32, mapId _map.Id, portalId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.sets.Remove(ctx, t, characterId, portalKey(mapId, portalId))
}

func (r *Registry) ClearForCharacter(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.sets.Clear(ctx, t, characterId)
}

func (r *Registry) GetForCharacter(ctx context.Context, characterId uint32) []Model {
	t := tenant.MustFromContext(ctx)
	members, err := r.sets.Members(ctx, t, characterId)
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
