package action

import (
	"context"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Kind* values identify which portal operation created a PendingAction, so the
// failure path can pick a message appropriate to what actually failed.
// The empty value means "written before Kind existed" and is treated as a
// transport, preserving the pre-task-184 message (task-184 FR-2.7).
const (
	KindWarp      = "warp"
	KindTransport = "transport"
)

// PendingAction represents a pending portal action awaiting saga completion
type PendingAction struct {
	CharacterId    uint32     `json:"characterId"`
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	FailureMessage string     `json:"failureMessage"`
	Kind           string     `json:"kind"`
}

// Registry tracks pending portal actions by saga ID
type Registry struct {
	reg *atlas.TenantRegistry[uuid.UUID, PendingAction]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uuid.UUID, PendingAction](client, "portal-action", func(k uuid.UUID) string {
			return k.String()
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

// Add registers a pending action for a saga
func (r *Registry) Add(ctx context.Context, sagaId uuid.UUID, a PendingAction) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Put(ctx, t, sagaId, a)
}

// AddWithTTL registers a pending action that self-expires. Add writes with no
// expiry, so a dropped COMPLETED event leaks the key forever; warp
// registrations use this instead. The TTL must comfortably exceed the saga's
// own timeout so the failure path can still find the entry.
func (r *Registry) AddWithTTL(ctx context.Context, sagaId uuid.UUID, a PendingAction, ttl time.Duration) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.PutWithTTL(ctx, t, sagaId, a, ttl)
}

// Get retrieves a pending action by saga ID
func (r *Registry) Get(ctx context.Context, sagaId uuid.UUID) (PendingAction, bool) {
	t := tenant.MustFromContext(ctx)
	v, err := r.reg.Get(ctx, t, sagaId)
	if err != nil {
		return PendingAction{}, false
	}
	return v, true
}

// Remove removes a pending action by saga ID
func (r *Registry) Remove(ctx context.Context, sagaId uuid.UUID) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Remove(ctx, t, sagaId)
}
