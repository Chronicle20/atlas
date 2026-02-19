package character

import (
	"atlas-buffs/buff"
	"atlas-buffs/buff/stat"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("not found")

type Registry struct {
	characters  *atlas.TenantRegistry[uint32, Model]
	poisonTicks *atlas.TenantRegistry[uint32, time.Time]
	client      *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		characters: atlas.NewTenantRegistry[uint32, Model](client, "buffs", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		poisonTicks: atlas.NewTenantRegistry[uint32, time.Time](client, "buffs-poison", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		client: client,
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) tenantSetKey() string {
	return "atlas:" + r.characters.Namespace() + ":_tenants"
}

func (r *Registry) Apply(ctx context.Context, worldId world.Id, channelId channel.Id, characterId uint32, sourceId int32, level byte, duration int32, changes []stat.Model) (buff.Model, error) {
	t := tenant.MustFromContext(ctx)

	b, err := buff.NewBuff(sourceId, level, duration, changes)
	if err != nil {
		return buff.Model{}, err
	}

	m, err := r.characters.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		m = Model{
			worldId:     worldId,
			channelId:   channelId,
			characterId: characterId,
			buffs:       make(map[int32]buff.Model),
		}
	} else if err != nil {
		return buff.Model{}, err
	} else {
		m.channelId = channelId
	}

	m.buffs[sourceId] = b
	err = r.characters.Put(ctx, t, characterId, m)
	if err != nil {
		return buff.Model{}, err
	}

	tb, _ := json.Marshal(&t)
	r.client.SAdd(ctx, r.tenantSetKey(), tb)

	return b, nil
}

func (r *Registry) Get(ctx context.Context, id uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, id)
	if errors.Is(err, atlas.ErrNotFound) {
		return Model{}, ErrNotFound
	}
	return m, err
}

func (r *Registry) GetTenants(ctx context.Context) ([]tenant.Model, error) {
	members, err := r.client.SMembers(ctx, r.tenantSetKey()).Result()
	if err != nil {
		return nil, err
	}
	var tenants []tenant.Model
	for _, mb := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(mb), &t); err != nil {
			continue
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (r *Registry) GetCharacters(ctx context.Context) []Model {
	t := tenant.MustFromContext(ctx)
	vals, err := r.characters.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}
	return vals
}

func (r *Registry) Cancel(ctx context.Context, characterId uint32, sourceId int32) (buff.Model, error) {
	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		return buff.Model{}, ErrNotFound
	}
	if err != nil {
		return buff.Model{}, err
	}

	var cancelled buff.Model
	var found bool
	not := make(map[int32]buff.Model)
	for id, b := range m.buffs {
		if b.SourceId() != sourceId {
			not[id] = b
		} else {
			cancelled = b
			found = true
		}
	}
	m.buffs = not
	_ = r.characters.Put(ctx, t, characterId, m)

	if !found {
		return buff.Model{}, ErrNotFound
	}
	return cancelled, nil
}

func (r *Registry) GetExpired(ctx context.Context, characterId uint32) []buff.Model {
	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return make([]buff.Model, 0)
	}

	not := make(map[int32]buff.Model)
	var expired []buff.Model
	for id, b := range m.buffs {
		if b.Expired() {
			expired = append(expired, b)
		} else {
			not[id] = b
		}
	}
	m.buffs = not
	_ = r.characters.Put(ctx, t, characterId, m)

	if len(expired) == 0 {
		return make([]buff.Model, 0)
	}
	return expired
}

func (r *Registry) CancelAll(ctx context.Context, characterId uint32) []buff.Model {
	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return make([]buff.Model, 0)
	}

	all := make([]buff.Model, 0, len(m.buffs))
	for _, b := range m.buffs {
		all = append(all, b)
	}
	m.buffs = make(map[int32]buff.Model)
	_ = r.characters.Put(ctx, t, characterId, m)

	return all
}

func (r *Registry) HasImmunity(ctx context.Context, characterId uint32) bool {
	t := tenant.MustFromContext(ctx)
	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return false
	}
	return hasImmunityBuff(m)
}

type PoisonTickEntry struct {
	Tenant      tenant.Model
	WorldId     world.Id
	ChannelId   channel.Id
	CharacterId uint32
	Amount      int32
}

func (r *Registry) GetPoisonCharacters(ctx context.Context) []PoisonTickEntry {
	t := tenant.MustFromContext(ctx)
	vals, err := r.characters.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}

	var results []PoisonTickEntry
	for _, m := range vals {
		for _, b := range m.buffs {
			if b.Expired() {
				continue
			}
			for _, c := range b.Changes() {
				if c.Type() == "POISON" {
					results = append(results, PoisonTickEntry{
						Tenant:      t,
						WorldId:     m.worldId,
						ChannelId:   m.channelId,
						CharacterId: m.characterId,
						Amount:      c.Amount(),
					})
					break
				}
			}
		}
	}
	return results
}

func (r *Registry) GetLastPoisonTick(ctx context.Context, characterId uint32) (time.Time, bool) {
	t := tenant.MustFromContext(ctx)
	tick, err := r.poisonTicks.Get(ctx, t, characterId)
	if err != nil {
		return time.Time{}, false
	}
	return tick, true
}

func (r *Registry) UpdatePoisonTick(ctx context.Context, characterId uint32, at time.Time) {
	t := tenant.MustFromContext(ctx)
	_ = r.poisonTicks.Put(ctx, t, characterId, at)
}

func (r *Registry) ClearPoisonTick(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.poisonTicks.Remove(ctx, t, characterId)
}
