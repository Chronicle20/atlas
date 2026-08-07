package action

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

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

// Add registers a pending action for a saga. A Redis write failure is logged
// (mirroring dedupe.Gate.Allow's idiom) rather than silently discarded: a
// dropped write here has the same "player recovers via a key that was never
// written" shape as AddWithTTL, just without the TTL urgency.
func (r *Registry) Add(l logrus.FieldLogger, ctx context.Context, sagaId uuid.UUID, a PendingAction) {
	t := tenant.MustFromContext(ctx)
	if err := r.reg.Put(ctx, t, sagaId, a); err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"tenant_id":      t.Id().String(),
			"transaction_id": sagaId.String(),
			"character_id":   a.CharacterId,
			"kind":           a.Kind,
		}).Warn("Failed to register pending portal action.")
	}
}

// AddWithTTL registers a pending action that self-expires. Add writes with no
// expiry, so a dropped COMPLETED event leaks the key forever; warp
// registrations use this instead. The TTL must comfortably exceed the saga's
// own timeout so the failure path can still find the entry.
//
// This registration is the SOLE recovery path for a warp whose saga never
// lands, because the ENTER handler stops emitting EnableActions once a warp
// is dispatched (task-184). A Redis write failure here must never be
// invisible: the failure path (kafka/consumer/saga/consumer.go
// handleStatusEventFailed) would find nothing and treat the saga as "not a
// portal action", leaving the player frozen with no trace.
func (r *Registry) AddWithTTL(l logrus.FieldLogger, ctx context.Context, sagaId uuid.UUID, a PendingAction, ttl time.Duration) {
	t := tenant.MustFromContext(ctx)
	if err := r.reg.PutWithTTL(ctx, t, sagaId, a, ttl); err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"tenant_id":      t.Id().String(),
			"transaction_id": sagaId.String(),
			"character_id":   a.CharacterId,
			"kind":           a.Kind,
		}).Warn("Failed to register pending portal action with TTL. A warp whose saga never lands will not be recovered.")
	}
}

// Get retrieves a pending action by saga ID. The boolean return is "found",
// exactly as before: callers that treat a Redis error as "not found" keep
// working unchanged. A genuine Redis error (as opposed to a clean cache miss)
// is additionally logged here, because callers cannot otherwise tell the two
// apart and a silent error on this read has the same frozen-player failure
// mode documented on AddWithTTL.
func (r *Registry) Get(l logrus.FieldLogger, ctx context.Context, sagaId uuid.UUID) (PendingAction, bool) {
	t := tenant.MustFromContext(ctx)
	v, err := r.reg.Get(ctx, t, sagaId)
	if err != nil {
		if !errors.Is(err, atlas.ErrNotFound) {
			l.WithError(err).WithFields(logrus.Fields{
				"tenant_id":      t.Id().String(),
				"transaction_id": sagaId.String(),
			}).Warn("Failed to read pending portal action. Treating as not found.")
		}
		return PendingAction{}, false
	}
	return v, true
}

// Remove removes a pending action by saga ID.
func (r *Registry) Remove(l logrus.FieldLogger, ctx context.Context, sagaId uuid.UUID) {
	t := tenant.MustFromContext(ctx)
	if err := r.reg.Remove(ctx, t, sagaId); err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"tenant_id":      t.Id().String(),
			"transaction_id": sagaId.String(),
		}).Warn("Failed to remove pending portal action.")
	}
}
