package channel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	channelConstant "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	client *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{client: client}
}

func getRegistry() *Registry {
	return registry
}

func channelSetKey(t tenant.Model) string {
	return fmt.Sprintf("transport:channels:%s", atlas.TenantKey(t))
}

func channelMember(ch channelConstant.Model) string {
	return fmt.Sprintf("%d:%d", ch.WorldId(), ch.Id())
}

func parseChannelMember(member string) (channelConstant.Model, bool) {
	parts := strings.SplitN(member, ":", 2)
	if len(parts) != 2 {
		return channelConstant.Model{}, false
	}
	worldId, err := strconv.Atoi(parts[0])
	if err != nil {
		return channelConstant.Model{}, false
	}
	channelId, err := strconv.Atoi(parts[1])
	if err != nil {
		return channelConstant.Model{}, false
	}
	return channelConstant.NewModel(world.Id(worldId), channelConstant.Id(channelId)), true
}

func (r *Registry) Add(ctx context.Context, model channelConstant.Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.client.SAdd(ctx, channelSetKey(t), channelMember(model)).Err()
}

func (r *Registry) Remove(ctx context.Context, ch channelConstant.Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.client.SRem(ctx, channelSetKey(t), channelMember(ch)).Err()
}

func (r *Registry) GetAll(ctx context.Context) []channelConstant.Model {
	t := tenant.MustFromContext(ctx)
	members, err := r.client.SMembers(ctx, channelSetKey(t)).Result()
	if err != nil {
		return nil
	}
	results := make([]channelConstant.Model, 0, len(members))
	for _, m := range members {
		if ch, ok := parseChannelMember(m); ok {
			results = append(results, ch)
		}
	}
	return results
}
