package action

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// PendingAction represents a pending portal action awaiting saga completion
type PendingAction struct {
	CharacterId    uint32     `json:"characterId"`
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	FailureMessage string     `json:"failureMessage"`
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
