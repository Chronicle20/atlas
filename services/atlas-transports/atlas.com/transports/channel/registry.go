package channel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	channelConstant "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	channels *atlas.TenantSet
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{channels: atlas.NewTenantSet(client, "transport:channels")}
}

func getRegistry() *Registry {
	return registry
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
	_ = r.channels.Add(ctx, t, channelMember(model))
}

func (r *Registry) Remove(ctx context.Context, ch channelConstant.Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.channels.Remove(ctx, t, channelMember(ch))
}

func (r *Registry) GetAll(ctx context.Context) []channelConstant.Model {
	t := tenant.MustFromContext(ctx)
	members, err := r.channels.Members(ctx, t)
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
