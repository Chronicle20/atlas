package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	channelConstant "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

const tenantSetKey = "channel:tenants"

type Registry struct {
	channels *atlas.TenantRegistry[string, Model]
	client   *goredis.Client
}

var channelRegistry *Registry

var ErrChannelNotFound = errors.New("channel not found")

func compositeKey(worldId world.Id, channelId channelConstant.Id) string {
	return fmt.Sprintf("%d:%d", worldId, channelId)
}

func InitRegistry(client *goredis.Client) {
	channelRegistry = &Registry{
		channels: atlas.NewTenantRegistry[string, Model](client, "channel", func(k string) string { return k }),
		client:   client,
	}
}

func GetChannelRegistry() *Registry {
	return channelRegistry
}

func (r *Registry) Register(ctx context.Context, m Model) Model {
	t := tenant.MustFromContext(ctx)
	key := compositeKey(m.worldId, m.channelId)
	_ = r.channels.Put(ctx, t, key, m)
	r.trackTenant(ctx, t)
	return m
}

func (r *Registry) ChannelServers(ctx context.Context) []Model {
	t := tenant.MustFromContext(ctx)
	vals, err := r.channels.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}
	return vals
}

func (r *Registry) ChannelServer(ctx context.Context, ch channelConstant.Model) (Model, error) {
	t := tenant.MustFromContext(ctx)
	key := compositeKey(ch.WorldId(), ch.Id())
	m, err := r.channels.Get(ctx, t, key)
	if err != nil {
		return Model{}, ErrChannelNotFound
	}
	return m, nil
}

func (r *Registry) RemoveByWorldAndChannel(ctx context.Context, ch channelConstant.Model) error {
	t := tenant.MustFromContext(ctx)
	key := compositeKey(ch.WorldId(), ch.Id())
	exists, _ := r.channels.Exists(ctx, t, key)
	if !exists {
		return ErrChannelNotFound
	}
	_ = r.channels.Remove(ctx, t, key)
	return nil
}

func (r *Registry) Tenants() []tenant.Model {
	ctx := context.Background()
	members, err := r.client.SMembers(ctx, tenantSetKey).Result()
	if err != nil {
		return nil
	}
	results := make([]tenant.Model, 0)
	for _, data := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			continue
		}
		results = append(results, t)
	}
	return results
}

func (r *Registry) trackTenant(ctx context.Context, t tenant.Model) {
	data, err := json.Marshal(&t)
	if err != nil {
		return
	}
	_ = r.client.SAdd(ctx, tenantSetKey, string(data)).Err()
}
