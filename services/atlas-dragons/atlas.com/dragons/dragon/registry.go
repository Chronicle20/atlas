package dragon

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// storedDragon is the JSON-serializable representation stored in Redis. The key
// carries the tenant id, but the tenant fields are carried in the value too so
// fromStored can rebuild a tenant, mirroring storedSummon in atlas-summons.
//
// X/Y are int32, not the int16 atlas-summons uses: SPAWN_DRAGON encodes 4-byte
// coordinates (CDragon::OnCreated Decode4 x, Decode4 y).
type storedDragon struct {
	TenantId           string `json:"tenantId"`
	TenantRegion       string `json:"tenantRegion"`
	TenantMajorVersion uint16 `json:"tenantMajorVersion"`
	TenantMinorVersion uint16 `json:"tenantMinorVersion"`
	OwnerCharacterId   uint32 `json:"ownerCharacterId"`
	WorldId            byte   `json:"worldId"`
	ChannelId          byte   `json:"channelId"`
	MapId              uint32 `json:"mapId"`
	Instance           string `json:"instance"`
	X                  int32  `json:"x"`
	Y                  int32  `json:"y"`
	Stance             byte   `json:"stance"`
	JobId              uint16 `json:"jobId"`
}

func toStored(t tenant.Model, m Model) storedDragon {
	f := m.Field()
	return storedDragon{
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		OwnerCharacterId:   m.OwnerCharacterId(),
		WorldId:            byte(f.WorldId()),
		ChannelId:          byte(f.ChannelId()),
		MapId:              uint32(f.MapId()),
		Instance:           f.Instance().String(),
		X:                  m.X(),
		Y:                  m.Y(),
		Stance:             m.Stance(),
		JobId:              uint16(m.JobId()),
	}
}

func fromStored(s storedDragon) (tenant.Model, Model, error) {
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
	m := NewBuilder(s.OwnerCharacterId).
		SetField(f).
		SetX(s.X).
		SetY(s.Y).
		SetStance(s.Stance).
		SetJobId(job.Id(s.JobId)).
		Build()
	return t, m, nil
}

// Registry is the authority for "which dragons exist and where". There is no id
// allocator and no owner index: the owner character id is the primary key, which
// makes "at most one dragon per character" a property of the key space rather
// than an invariant to enforce.
type Registry struct {
	reg      *atlasredis.Registry[string, storedDragon]
	fieldIdx *atlasredis.KeyedSet[string]
}

var (
	registry *Registry
	once     sync.Once
)

func newRegistry(rc *goredis.Client) *Registry {
	return &Registry{
		reg:      atlasredis.NewRegistry[string, storedDragon](rc, "dragon", func(s string) string { return s }),
		fieldIdx: atlasredis.NewKeyedSet[string](rc, "dragon-map", func(s string) string { return s }),
	}
}

// InitRegistry initializes the package-level Registry singleton. Safe to call
// more than once; only the first call takes effect.
func InitRegistry(rc *goredis.Client) { once.Do(func() { registry = newRegistry(rc) }) }

// GetRegistry returns the package-level Registry singleton set up by InitRegistry.
func GetRegistry() *Registry { return registry }

func storeSuffix(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), characterId)
}

func fieldSuffix(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

func member(characterId uint32) string { return fmt.Sprintf("%d", characterId) }

// Put stores m under its owner character id, overwriting any prior dragon for
// that character. If the character already had a dragon on a different field,
// the stale field-index membership is cleaned up first so the dragon never
// appears in two fields' indexes at once.
func (r *Registry) Put(ctx context.Context, t tenant.Model, m Model) error {
	if prev, err := r.Get(ctx, t, m.OwnerCharacterId()); err == nil {
		if prev.Field() != m.Field() {
			_ = r.fieldIdx.Remove(ctx, fieldSuffix(t, prev.Field()), member(m.OwnerCharacterId()))
		}
	}
	if err := r.reg.Put(ctx, storeSuffix(t, m.OwnerCharacterId()), toStored(t, m)); err != nil {
		return err
	}
	return r.fieldIdx.Add(ctx, fieldSuffix(t, m.Field()), member(m.OwnerCharacterId()))
}

func (r *Registry) Get(ctx context.Context, t tenant.Model, characterId uint32) (Model, error) {
	s, err := r.reg.Get(ctx, storeSuffix(t, characterId))
	if err != nil {
		return Model{}, err
	}
	_, m, derr := fromStored(s)
	if derr != nil {
		return Model{}, derr
	}
	return m, nil
}

// Exists reports whether characterId currently owns a dragon for this tenant.
func (r *Registry) Exists(ctx context.Context, t tenant.Model, characterId uint32) (bool, error) {
	return r.reg.Exists(ctx, storeSuffix(t, characterId))
}

// GetInField returns every dragon currently indexed on field f for this tenant.
// Stale index entries (dragon removed without going through Remove) are
// skipped rather than surfaced as an error.
func (r *Registry) GetInField(ctx context.Context, t tenant.Model, f field.Model) ([]Model, error) {
	members, err := r.fieldIdx.Members(ctx, fieldSuffix(t, f))
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(members))
	for _, mem := range members {
		var characterId uint32
		if _, err := fmt.Sscanf(mem, "%d", &characterId); err != nil {
			continue
		}
		m, err := r.Get(ctx, t, characterId)
		if err != nil {
			continue // stale index entry
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *Registry) Update(ctx context.Context, t tenant.Model, characterId uint32, fn func(Model) Model) (Model, error) {
	s, err := r.reg.Update(ctx, storeSuffix(t, characterId), func(cur storedDragon) storedDragon {
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

// Remove deletes the dragon and reports whether one existed. The bool is what
// makes destroy idempotent at the processor level: no dragon means no DESTROYED
// event, not an error.
func (r *Registry) Remove(ctx context.Context, t tenant.Model, characterId uint32) (bool, error) {
	m, err := r.Get(ctx, t, characterId)
	if err == nil {
		_ = r.fieldIdx.Remove(ctx, fieldSuffix(t, m.Field()), member(characterId))
	}
	return r.reg.RemoveExisting(ctx, storeSuffix(t, characterId))
}
