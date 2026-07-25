package character

import (
	"atlas-buffs/buff"
	"atlas-buffs/buff/stat"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var ErrNotFound = errors.New("not found")

type Registry struct {
	characters  *atlas.TenantRegistry[uint32, Model]
	poisonTicks *atlas.TenantRegistry[uint32, time.Time]
	tenants     *atlas.Set
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
		tenants: atlas.NewSet(client, "buffs:_tenants"),
	}
}

func GetRegistry() *Registry {
	return registry
}

// srcKey is the map key for a normal whole-source buff (replace-on-recast).
func srcKey(sourceId int32) string {
	return strconv.FormatInt(int64(sourceId), 10)
}

// statKey is the map key for an accumulate-mode per-stat buff: each stat of a
// source is tracked under its own key so it carries an independent timer and
// expires on its own (Beholder Hex). Re-applying the same stat overwrites just
// that key (timer refresh); a different stat of the same source coexists.
func statKey(sourceId int32, statType string) string {
	return srcKey(sourceId) + ":" + statType
}

// Apply stores a buff for characterId and returns the buff(s) created so the
// caller can emit one APPLIED event each.
//
// accumulate == false (default): the whole source is one buff keyed by sourceId;
// a re-apply replaces it (refresh). Returns exactly one buff.
//
// accumulate == true: each change is stored as its own single-stat buff keyed by
// (sourceId, statType), each with its own expiry; other stats of the same source
// are left intact, so the source's buffs accumulate one-at-a-time. Returns one
// buff per change.
func (r *Registry) Apply(ctx context.Context, worldId world.Id, channelId channel.Id, characterId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool) ([]buff.Model, error) {
	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		m = Model{
			worldId:     worldId,
			channelId:   channelId,
			characterId: characterId,
			buffs:       make(map[string]buff.Model),
		}
	} else if err != nil {
		return nil, err
	} else {
		m.channelId = channelId
	}

	var applied []buff.Model
	if accumulate {
		for _, c := range changes {
			b, err := buff.NewBuff(sourceId, level, duration, []stat.Model{c})
			if err != nil {
				return nil, err
			}
			m.buffs[statKey(sourceId, c.Type())] = b
			applied = append(applied, b)
		}
	} else {
		b, err := buff.NewBuff(sourceId, level, duration, changes)
		if err != nil {
			return nil, err
		}
		m.buffs[srcKey(sourceId)] = b
		applied = append(applied, b)
	}

	if err := r.characters.Put(ctx, t, characterId, m); err != nil {
		return nil, err
	}

	if tb, err := json.Marshal(&t); err == nil {
		_ = r.tenants.Add(ctx, string(tb))
	}

	return applied, nil
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
	members, err := r.tenants.Members(ctx)
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

// Cancel removes every buff for the character whose SourceId matches and returns
// them all, so the caller can emit one EXPIRED per removed buff. In accumulate
// mode a single sourceId (e.g. Beholder Hex 1320009) maps to several per-stat
// buffs; returning only one would leave the other stats' icons stuck on the
// client (removed from storage but never cancelled). Returns ErrNotFound when no
// buff matched.
func (r *Registry) Cancel(ctx context.Context, characterId uint32, sourceId int32) ([]buff.Model, error) {
	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	cancelled := make([]buff.Model, 0)
	not := make(map[string]buff.Model)
	for id, b := range m.buffs {
		if b.SourceId() != sourceId {
			not[id] = b
		} else {
			cancelled = append(cancelled, b)
		}
	}
	m.buffs = not
	_ = r.characters.Put(ctx, t, characterId, m)

	if len(cancelled) == 0 {
		return nil, ErrNotFound
	}
	return cancelled, nil
}

func (r *Registry) GetExpired(ctx context.Context, characterId uint32) []buff.Model {
	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if err != nil {
		return make([]buff.Model, 0)
	}

	not := make(map[string]buff.Model)
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
	m.buffs = make(map[string]buff.Model)
	_ = r.characters.Put(ctx, t, characterId, m)

	return all
}

// CancelByStatTypes removes any buff whose Changes() intersects typeSet.
// Returns the cancelled buffs (caller emits EXPIRED events).
// Empty typeSet returns (nil, nil) without touching Redis.
func (r *Registry) CancelByStatTypes(ctx context.Context, characterId uint32, typeSet map[string]bool) ([]buff.Model, error) {
	if len(typeSet) == 0 {
		return nil, nil
	}

	t := tenant.MustFromContext(ctx)

	m, err := r.characters.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cancelled := make([]buff.Model, 0)
	keep := make(map[string]buff.Model)
	for id, b := range m.buffs {
		matched := false
		for _, c := range b.Changes() {
			if typeSet[c.Type()] {
				matched = true
				break
			}
		}
		if matched {
			cancelled = append(cancelled, b)
		} else {
			keep[id] = b
		}
	}

	if len(cancelled) == 0 {
		return nil, nil
	}

	m.buffs = keep
	if err := r.characters.Put(ctx, t, characterId, m); err != nil {
		return nil, err
	}
	return cancelled, nil
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
