// Package dedupe gates duplicate portal ENTER commands.
//
// The GMS v83 client re-checks portal collision every frame while the player
// overlaps a portal's rect (CUserLocal::CheckPortal_Collision), and the
// scripted-portal path has no minimum re-send interval — unlike
// CField::SendTransferFieldRequest, which refuses to re-send within 500ms.
// The only thing stopping a re-send is m_bExclRequestSent, which the server
// clears via EnableActions. task-184's primary fix is to stop clearing that
// flag while a warp is in flight; this gate is defence in depth against any
// future unlock-shaped regression in an outcome path nobody is looking at.
//
// It fails OPEN: losing Redis must not make every portal in the game unusable.
package dedupe

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// enterGateTTL is how long one portal ENTER closes the gate for.
// Comfortably above the client's own 500ms floor for non-scripted portals, and
// comfortably below any interval at which a player could legitimately intend
// to re-enter the same portal in the same map instance (task-184 FR-3.4).
const enterGateTTL = 2 * time.Second

// lockNamespace is the Redis key namespace for the gate. Disjoint from every
// existing namespace in this service.
const lockNamespace = "portal-enter"

// Key identifies one portal entry attempt. Tenant is NOT a field: it comes
// from the context via the standard libs/atlas-redis tenant key helper.
type Key struct {
	CharacterId uint32
	MapId       _map.Id
	Instance    uuid.UUID
	PortalId    uint32
}

// Gate decides whether a portal ENTER should be processed.
type Gate interface {
	// Allow reports whether this ENTER should be processed. A duplicate inside
	// the TTL window returns false. Any Redis error returns true (fail open).
	Allow(l logrus.FieldLogger, ctx context.Context, k Key) bool
}

type redisGate struct {
	lock *atlas.Lock
}

// nilGate is returned when the gate was never initialised (unit tests, or a
// startup path that skipped InitGate). It allows everything.
type nilGate struct{}

func (nilGate) Allow(_ logrus.FieldLogger, _ context.Context, _ Key) bool { return true }

var gate Gate

// InitGate wires the gate to Redis. Call once at startup, beside
// action.InitRegistry.
func InitGate(client *goredis.Client) {
	gate = &redisGate{lock: atlas.NewLock(client, lockNamespace)}
}

// GetGate returns the process gate, or a permissive gate if InitGate was never
// called. It never returns nil, so callers need no nil check (FR-3.6).
func GetGate() Gate {
	if gate == nil {
		return nilGate{}
	}
	return gate
}

// redisKey composes the tenant-scoped key. Lock is not tenant-aware — its
// lockKey is namespacedKey(namespace, "_lock", key) with no tenant segment —
// so the tenant is composed into the caller-supplied key using the library's
// own TenantKey and CompositeKey helpers rather than hand-rolled string
// concatenation (task-184 FR-3.3, design §5.1).
func redisKey(t tenant.Model, k Key) string {
	return atlas.CompositeKey(
		atlas.TenantKey(t),
		strconv.FormatUint(uint64(k.CharacterId), 10),
		strconv.FormatUint(uint64(k.MapId), 10),
		k.Instance.String(),
		strconv.FormatUint(uint64(k.PortalId), 10),
	)
}

func (g *redisGate) Allow(l logrus.FieldLogger, ctx context.Context, k Key) bool {
	t := tenant.MustFromContext(ctx)
	rk := redisKey(t, k)

	// The lock is never released — TTL expiry IS the release. A successful
	// portal entry keeps the gate closed for the full window.
	acquired, err := g.lock.AcquireWithTTL(ctx, rk, enterGateTTL)
	if err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"tenant_id":    t.Id().String(),
			"character_id": k.CharacterId,
			"portal_id":    k.PortalId,
		}).Warn("Portal enter dedupe gate unavailable, processing command. Duplicate ENTER commands are not being suppressed.")
		return true
	}
	if !acquired {
		l.WithFields(logrus.Fields{
			"tenant_id":    t.Id().String(),
			"character_id": k.CharacterId,
			"map_id":       uint32(k.MapId),
			"instance":     k.Instance.String(),
			"portal_id":    k.PortalId,
		}).Debug("Dropping duplicate portal enter command inside the dedupe window.")
		return false
	}
	return true
}
