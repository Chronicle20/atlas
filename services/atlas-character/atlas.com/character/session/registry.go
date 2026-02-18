package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg    *atlas.TenantRegistry[uint32, Model]
	client *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, Model](client, "character-session", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		client: client,
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) tenantSetKey() string {
	return fmt.Sprintf("atlas:%s:_tenants", r.reg.Namespace())
}

func (r *Registry) Add(ctx context.Context, characterId uint32, ch channel.Model, state State) error {
	t := tenant.MustFromContext(ctx)

	existing, err := r.reg.Get(ctx, t, characterId)
	if err == nil && existing.State() == StateLoggedIn {
		return errors.New("already logged in")
	}

	m := Model{
		tenant:      t,
		characterId: characterId,
		worldId:     ch.WorldId(),
		channelId:   ch.Id(),
		state:       state,
		age:         time.Now(),
	}

	err = r.reg.Put(ctx, t, characterId, m)
	if err != nil {
		return err
	}

	tb, _ := json.Marshal(&t)
	r.client.SAdd(ctx, r.tenantSetKey(), tb)
	return nil
}

func (r *Registry) Set(ctx context.Context, characterId uint32, ch channel.Model, state State) error {
	t := tenant.MustFromContext(ctx)

	m := Model{
		tenant:      t,
		characterId: characterId,
		worldId:     ch.WorldId(),
		channelId:   ch.Id(),
		state:       state,
		age:         time.Now(),
	}

	err := r.reg.Put(ctx, t, characterId, m)
	if err != nil {
		return err
	}

	tb, _ := json.Marshal(&t)
	r.client.SAdd(ctx, r.tenantSetKey(), tb)
	return nil
}

func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	return r.reg.Get(ctx, t, characterId)
}

func (r *Registry) GetAll(ctx context.Context) []Model {
	members, err := r.client.SMembers(ctx, r.tenantSetKey()).Result()
	if err != nil {
		return nil
	}

	var results []Model
	for _, mb := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(mb), &t); err != nil {
			continue
		}
		vals, err := r.reg.GetAllValues(ctx, t)
		if err != nil {
			continue
		}
		results = append(results, vals...)
	}
	return results
}

func (r *Registry) Remove(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Remove(ctx, t, characterId)
}
