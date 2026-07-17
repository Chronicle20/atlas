package summon

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// storedSummon is the JSON-serializable representation stored in Redis. The
// keys carry the tenant id, so it is not duplicated in the value; the full
// field (world/channel/map/instance) IS carried so fromStored can rebuild a
// field.Model. Times serialize as unix-milli (0 == zero time), matching the
// monster blueprint's time-field convention.
type storedSummon struct {
	Id                 uint32       `json:"id"`
	TenantId           string       `json:"tenantId"`
	TenantRegion       string       `json:"tenantRegion"`
	TenantMajorVersion uint16       `json:"tenantMajorVersion"`
	TenantMinorVersion uint16       `json:"tenantMinorVersion"`
	OwnerCharacterId   uint32       `json:"ownerCharacterId"`
	SkillId            uint32       `json:"skillId"`
	SkillLevel         byte         `json:"skillLevel"`
	SummonType         string       `json:"summonType"`
	MovementType       byte         `json:"movementType"`
	WorldId            byte         `json:"worldId"`
	ChannelId          byte         `json:"channelId"`
	MapId              uint32       `json:"mapId"`
	Instance           string       `json:"instance"`
	X                  int16        `json:"x"`
	Y                  int16        `json:"y"`
	Stance             byte         `json:"stance"`
	Hp                 int32        `json:"hp"`
	MaxHp              int32        `json:"maxHp"`
	Animated           bool         `json:"animated"`
	SpawnTimeMs        int64        `json:"spawnTimeMs,omitempty"`
	ExpiresAtMs        int64        `json:"expiresAtMs,omitempty"`
	NextHealAtMs       int64        `json:"nextHealAtMs,omitempty"`
	NextBuffAtMs       int64        `json:"nextBuffAtMs,omitempty"`
	HealAmount         int16        `json:"healAmount,omitempty"`
	HealIntervalMs     int64        `json:"healIntervalMs,omitempty"`
	BuffIntervalMs     int64        `json:"buffIntervalMs,omitempty"`
	BuffSourceId       int32        `json:"buffSourceId,omitempty"`
	BuffLevel          byte         `json:"buffLevel,omitempty"`
	BuffDuration       int32        `json:"buffDuration,omitempty"`
	BuffChanges        []StatChange `json:"buffChanges,omitempty"`
}

// timeToMs returns 0 for the zero time, else unix-milli.
func timeToMs(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// msToTime returns the zero time for 0, else the unix-milli instant (UTC).
func msToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func toStored(t tenant.Model, m Model) storedSummon {
	changes := make([]StatChange, len(m.buffChanges))
	copy(changes, m.buffChanges)
	f := m.Field()
	return storedSummon{
		Id:                 m.id,
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		OwnerCharacterId:   m.ownerCharacterId,
		SkillId:            m.skillId,
		SkillLevel:         m.skillLevel,
		SummonType:         string(m.summonType),
		MovementType:       byte(m.movementType),
		WorldId:            byte(f.WorldId()),
		ChannelId:          byte(f.ChannelId()),
		MapId:              uint32(f.MapId()),
		Instance:           f.Instance().String(),
		X:                  m.x,
		Y:                  m.y,
		Stance:             m.stance,
		Hp:                 m.hp,
		MaxHp:              m.maxHp,
		Animated:           m.animated,
		SpawnTimeMs:        timeToMs(m.spawnTime),
		ExpiresAtMs:        timeToMs(m.expiresAt),
		NextHealAtMs:       timeToMs(m.nextHealAt),
		NextBuffAtMs:       timeToMs(m.nextBuffAt),
		HealAmount:         m.healAmount,
		HealIntervalMs:     m.healInterval.Milliseconds(),
		BuffIntervalMs:     m.buffInterval.Milliseconds(),
		BuffSourceId:       m.buffSourceId,
		BuffLevel:          m.buffLevel,
		BuffDuration:       m.buffDuration,
		BuffChanges:        changes,
	}
}

func fromStored(s storedSummon) (tenant.Model, Model, error) {
	tenantId, err := uuid.Parse(s.TenantId)
	if err != nil {
		return tenant.Model{}, Model{}, err
	}
	t, err := tenant.Create(tenantId, s.TenantRegion, s.TenantMajorVersion, s.TenantMinorVersion)
	if err != nil {
		return tenant.Model{}, Model{}, err
	}
	inst, perr := uuid.Parse(s.Instance)
	if perr != nil {
		inst = uuid.Nil
	}
	f := field.NewBuilder(world.Id(s.WorldId), channel.Id(s.ChannelId), _map.Id(s.MapId)).
		SetInstance(inst).Build()
	m := NewBuilder().
		SetId(s.Id).
		SetOwnerCharacterId(s.OwnerCharacterId).
		SetSkillId(s.SkillId).
		SetSkillLevel(s.SkillLevel).
		SetSummonType(SummonType(s.SummonType)).
		SetMovementType(MovementType(s.MovementType)).
		SetField(f).
		SetX(s.X).
		SetY(s.Y).
		SetStance(s.Stance).
		SetHp(s.Hp).
		SetMaxHp(s.MaxHp).
		SetAnimated(s.Animated).
		SetSpawnTime(msToTime(s.SpawnTimeMs)).
		SetExpiresAt(msToTime(s.ExpiresAtMs)).
		SetNextHealAt(msToTime(s.NextHealAtMs)).
		SetNextBuffAt(msToTime(s.NextBuffAtMs)).
		SetHealAmount(s.HealAmount).
		SetHealInterval(time.Duration(s.HealIntervalMs) * time.Millisecond).
		SetBuffInterval(time.Duration(s.BuffIntervalMs) * time.Millisecond).
		SetBuffSourceId(s.BuffSourceId).
		SetBuffLevel(s.BuffLevel).
		SetBuffDuration(s.BuffDuration).
		SetBuffChanges(s.BuffChanges).
		Build()
	return t, m, nil
}

type Registry struct {
	reg      *atlasredis.Registry[string, storedSummon]
	fieldIdx *atlasredis.KeyedSet[string]
	ownerIdx *atlasredis.KeyedSet[string]
}

var (
	registry *Registry
	once     sync.Once
)

func newRegistry(rc *goredis.Client) *Registry {
	return &Registry{
		reg:      atlasredis.NewRegistry[string, storedSummon](rc, "summon", func(s string) string { return s }),
		fieldIdx: atlasredis.NewKeyedSet[string](rc, "summon-map", func(s string) string { return s }),
		ownerIdx: atlasredis.NewKeyedSet[string](rc, "summon-owner", func(s string) string { return s }),
	}
}

func InitRegistry(rc *goredis.Client) { once.Do(func() { registry = newRegistry(rc) }) }
func GetRegistry() *Registry          { return registry }

func storeSuffix(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), id)
}

func fieldSuffix(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

func ownerSuffix(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), characterId)
}

func (r *Registry) Put(ctx context.Context, t tenant.Model, m Model) error {
	if err := r.reg.Put(ctx, storeSuffix(t, m.Id()), toStored(t, m)); err != nil {
		return err
	}
	member := fmt.Sprintf("%d", m.Id())
	if err := r.fieldIdx.Add(ctx, fieldSuffix(t, m.Field()), member); err != nil {
		return err
	}
	return r.ownerIdx.Add(ctx, ownerSuffix(t, m.OwnerCharacterId()), member)
}

func (r *Registry) Get(ctx context.Context, t tenant.Model, id uint32) (Model, error) {
	s, err := r.reg.Get(ctx, storeSuffix(t, id))
	if err != nil {
		return Model{}, err
	}
	_, m, derr := fromStored(s)
	if derr != nil {
		return Model{}, derr
	}
	return m, nil
}

func (r *Registry) GetInField(ctx context.Context, t tenant.Model, f field.Model) ([]Model, error) {
	return r.loadMembers(ctx, t, r.fieldIdx, fieldSuffix(t, f))
}

func (r *Registry) GetByOwner(ctx context.Context, t tenant.Model, characterId uint32) ([]Model, error) {
	return r.loadMembers(ctx, t, r.ownerIdx, ownerSuffix(t, characterId))
}

func (r *Registry) loadMembers(ctx context.Context, t tenant.Model, set *atlasredis.KeyedSet[string], key string) ([]Model, error) {
	members, err := set.Members(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(members))
	for _, member := range members {
		var id uint32
		if _, err := fmt.Sscanf(member, "%d", &id); err != nil {
			continue
		}
		m, err := r.Get(ctx, t, id)
		if err != nil {
			// stale index entry; skip
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *Registry) Update(ctx context.Context, t tenant.Model, id uint32, fn func(Model) Model) (Model, error) {
	s, err := r.reg.Update(ctx, storeSuffix(t, id), func(cur storedSummon) storedSummon {
		_, m, derr := fromStored(cur)
		if derr != nil {
			return cur
		}
		return toStored(t, fn(m))
	})
	if err != nil {
		return Model{}, err
	}
	_, m, derr := fromStored(s)
	if derr != nil {
		return Model{}, derr
	}
	return m, nil
}

func (r *Registry) Remove(ctx context.Context, t tenant.Model, id uint32) error {
	m, err := r.Get(ctx, t, id)
	if err == nil {
		member := fmt.Sprintf("%d", id)
		_ = r.fieldIdx.Remove(ctx, fieldSuffix(t, m.Field()), member)
		_ = r.ownerIdx.Remove(ctx, ownerSuffix(t, m.OwnerCharacterId()), member)
	}
	return r.reg.Remove(ctx, storeSuffix(t, id))
}

// GetAll returns every stored summon grouped by tenant. The tenant is rebuilt
// from the fields embedded in each stored value (mirroring atlas-monsters'
// Registry.GetMonsters), so sweep tasks can construct a tenant-scoped context
// per group. Undecodable entries are skipped.
func (r *Registry) GetAll(ctx context.Context) (map[tenant.Model][]Model, error) {
	stored, err := r.reg.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[tenant.Model][]Model)
	for _, s := range stored {
		t, m, derr := fromStored(s)
		if derr != nil {
			continue
		}
		out[t] = append(out[t], m)
	}
	return out, nil
}
