package kite

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// entry is the serialised form of Model. TenantRegistry marshals its value to
// JSON, so the stored shape needs exported fields; Model stays immutable.
type entry struct {
	Id          uint32     `json:"id"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Name        string     `json:"name"`
	TemplateId  uint32     `json:"templateId"`
	Message     string     `json:"message"`
	X           int16      `json:"x"`
	Y           int16      `json:"y"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Registry struct {
	reg  *atlas.TenantRegistry[uint32, entry]
	ids  *atlas.IDGenerator
	lock *atlas.Lock
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, entry](client, "kite", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		ids:  atlas.NewIDGenerator(client, "kite"),
		lock: atlas.NewLock(client, "kite-cap"),
	}
}

func getRegistry() *Registry {
	return registry
}

func toEntry(m Model) entry {
	return entry{
		Id:          m.Id(),
		WorldId:     m.Field().WorldId(),
		ChannelId:   m.Field().ChannelId(),
		MapId:       m.Field().MapId(),
		Instance:    m.Field().Instance(),
		CharacterId: m.CharacterId(),
		Name:        m.Name(),
		TemplateId:  m.TemplateId(),
		Message:     m.Message(),
		X:           m.X(),
		Y:           m.Y(),
		CreatedAt:   m.CreatedAt(),
	}
}

func fromEntry(e entry) Model {
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	return NewBuilder(e.Id, f, e.CharacterId).
		SetName(e.Name).
		SetTemplateId(e.TemplateId).
		SetMessage(e.Message).
		SetPosition(e.X, e.Y).
		SetCreatedAt(e.CreatedAt).
		Build()
}

func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	e, err := r.reg.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, false
	}
	return fromEntry(e), true
}

func (r *Registry) Put(ctx context.Context, m Model) error {
	t := tenant.MustFromContext(ctx)
	return r.reg.Put(ctx, t, m.CharacterId(), toEntry(m))
}

func (r *Registry) Remove(ctx context.Context, characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	return r.reg.Remove(ctx, t, characterId)
}

func (r *Registry) Exists(ctx context.Context, characterId uint32) (bool, error) {
	t := tenant.MustFromContext(ctx)
	return r.reg.Exists(ctx, t, characterId)
}

// NextId allocates the wire id. It is a tenant-scoped Redis INCR, not a
// process-local counter, because REMOVE_KITE addresses a kite by this id alone
// and any atlas-kites replica may be the one that allocated it.
func (r *Registry) NextId(ctx context.Context) (uint32, error) {
	return r.ids.NextID(ctx, tenant.MustFromContext(ctx))
}

// fieldLockKey includes the tenant because atlas.Lock is NOT tenant-scoped —
// it namespaces by the constructor's namespace only.
func fieldLockKey(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

// AcquireFieldLock serialises {count -> validate -> allocate -> insert} for one
// field. The per-character invariant is already safe (the command topic is
// keyed on characterId, so one character's commands share a partition), but the
// per-map cap is not: two different characters placing on the same
// full-but-for-one map land on different partitions.
func (r *Registry) AcquireFieldLock(ctx context.Context, f field.Model) (bool, error) {
	return r.lock.Acquire(ctx, fieldLockKey(tenant.MustFromContext(ctx), f))
}

func (r *Registry) ReleaseFieldLock(ctx context.Context, f field.Model) error {
	return r.lock.Release(ctx, fieldLockKey(tenant.MustFromContext(ctx), f))
}
