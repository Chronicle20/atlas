package messenger

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	messengers *atlas.TenantRegistry[uint32, Model]
	idGen      *atlas.IDGenerator
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		messengers: atlas.NewTenantRegistry[uint32, Model](client, "messenger", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		idGen: atlas.NewIDGenerator(client, "messenger"),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Create(ctx context.Context, characterId uint32) Model {
	t := tenant.MustFromContext(ctx)

	messengerId, _ := r.idGen.NextID(ctx, t)

	m, _ := NewBuilder().
		SetTenantId(t.Id()).
		SetId(messengerId).
		AddMember(characterId, 0).
		Build()

	_ = r.messengers.Put(ctx, t, messengerId, m)
	return m
}

func (r *Registry) GetAll(ctx context.Context) []Model {
	t := tenant.MustFromContext(ctx)

	results, err := r.messengers.GetAllValues(ctx, t)
	if err != nil {
		return make([]Model, 0)
	}
	return results
}

func (r *Registry) Get(ctx context.Context, messengerId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	return r.messengers.Get(ctx, t, messengerId)
}

func (r *Registry) Update(ctx context.Context, id uint32, updaters ...func(m Model) Model) (Model, error) {
	t := tenant.MustFromContext(ctx)

	m, err := r.messengers.Get(ctx, t, id)
	if err != nil {
		return Model{}, err
	}

	for _, updater := range updaters {
		m = updater(m)
	}

	if len(m.members) > MaxMembers {
		return Model{}, ErrAtCapacity
	}

	err = r.messengers.Put(ctx, t, id, m)
	if err != nil {
		return Model{}, err
	}
	return m, nil
}

func (r *Registry) Remove(ctx context.Context, messengerId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.messengers.Remove(ctx, t, messengerId)
}
