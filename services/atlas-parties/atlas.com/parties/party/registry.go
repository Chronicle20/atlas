package party

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	parties        *atlas.TenantRegistry[uint32, Model]
	idGen          *atlas.IDGenerator
	characterIndex *atlas.Uint32Index
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		parties: atlas.NewTenantRegistry[uint32, Model](client, "party", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		idGen:          atlas.NewIDGenerator(client, "party"),
		characterIndex: atlas.NewUint32Index(client, "party", "char-party"),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Create(ctx context.Context, leaderId uint32) Model {
	t := tenant.MustFromContext(ctx)
	partyId, _ := r.idGen.NextID(ctx, t)

	m := Model{
		tenantId: t.Id(),
		id:       partyId,
		leaderId: leaderId,
		members:  []uint32{leaderId},
	}
	_ = r.parties.Put(ctx, t, partyId, m)
	_ = r.characterIndex.Add(ctx, t, leaderId, partyId)
	return m
}

func (r *Registry) GetAll(ctx context.Context) []Model {
	t := tenant.MustFromContext(ctx)
	vals, err := r.parties.GetAllValues(ctx, t)
	if err != nil {
		return make([]Model, 0)
	}
	return vals
}

func (r *Registry) Get(ctx context.Context, partyId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	return r.parties.Get(ctx, t, partyId)
}

func (r *Registry) Update(ctx context.Context, id uint32, updaters ...func(m Model) Model) (Model, error) {
	t := tenant.MustFromContext(ctx)
	oldModel, err := r.parties.Get(ctx, t, id)
	if err != nil {
		return Model{}, err
	}

	newModel := oldModel
	for _, updater := range updaters {
		newModel = updater(newModel)
	}

	if len(newModel.members) > 6 {
		return Model{}, ErrAtCapacity
	}

	r.updateCharacterIndex(ctx, t, oldModel, newModel)
	_ = r.parties.Put(ctx, t, id, newModel)
	return newModel, nil
}

func (r *Registry) Remove(ctx context.Context, partyId uint32) {
	t := tenant.MustFromContext(ctx)
	party, err := r.parties.Get(ctx, t, partyId)
	if err == nil {
		for _, memberId := range party.members {
			_ = r.characterIndex.Remove(ctx, t, memberId, partyId)
		}
	}
	_ = r.parties.Remove(ctx, t, partyId)
}

func (r *Registry) updateCharacterIndex(ctx context.Context, t tenant.Model, oldModel, newModel Model) {
	oldMembers := make(map[uint32]bool)
	for _, memberId := range oldModel.members {
		oldMembers[memberId] = true
	}
	newMembers := make(map[uint32]bool)
	for _, memberId := range newModel.members {
		newMembers[memberId] = true
	}
	for memberId := range oldMembers {
		if !newMembers[memberId] {
			_ = r.characterIndex.Remove(ctx, t, memberId, oldModel.id)
		}
	}
	for memberId := range newMembers {
		if !oldMembers[memberId] {
			_ = r.characterIndex.Add(ctx, t, memberId, newModel.id)
		}
	}
}

func (r *Registry) GetPartyByCharacter(ctx context.Context, characterId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	partyId, err := r.characterIndex.LookupOne(ctx, t, characterId)
	if err != nil {
		return Model{}, ErrNotFound
	}
	return r.parties.Get(ctx, t, partyId)
}
