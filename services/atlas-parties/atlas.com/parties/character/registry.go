package character

import (
	"context"
	"errors"
	"strconv"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("not found")

type Registry struct {
	characters *atlas.TenantRegistry[uint32, Model]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		characters: atlas.NewTenantRegistry[uint32, Model](client, "party-character", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Create(ctx context.Context, f field.Model, id uint32, name string, level byte, jobId job.Id, gm int) Model {
	t := tenant.MustFromContext(ctx)
	m := Model{
		tenantId: t.Id(),
		id:       id,
		name:     name,
		level:    level,
		jobId:    jobId,
		field:    f,
		partyId:  0,
		online:   false,
		gm:       gm,
	}
	_ = r.characters.Put(ctx, t, id, m)
	return m
}

func (r *Registry) Get(ctx context.Context, id uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, id)
	if err != nil {
		return Model{}, ErrNotFound
	}
	return m, nil
}

func (r *Registry) Update(ctx context.Context, id uint32, updaters ...func(m Model) Model) Model {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, id)
	if err != nil {
		return Model{}
	}
	for _, updater := range updaters {
		m = updater(m)
	}
	_ = r.characters.Put(ctx, t, id, m)
	return m
}

func (r *Registry) Delete(ctx context.Context, id uint32) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.characters.Get(ctx, t, id)
	if err != nil {
		return ErrNotFound
	}
	return r.characters.Remove(ctx, t, id)
}
