package map_

import (
	"context"
	"sync"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(mapId _map.Id) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

// cacheKey scopes the per-map cache by tenant. atlas-data's reader pulls
// tenant-scoped string registries (place/street name) and the libs/atlas-rest
// pipeline auto-attaches a tenant header on the underlying GET, so the
// returned Model is tenant-specific in principle. Foothold geometry happens
// to be tenant-invariant in this deployment, but keying by (tenant, mapId)
// keeps the cache correct if that ever changes.
type cacheKey struct {
	tenantId uuid.UUID
	mapId    _map.Id
}

// Map data is static once loaded (foothold tree, return map, etc. don't
// change at runtime), so we cache forever process-wide.
var (
	mapCache  sync.Map // map[cacheKey]Model
	mapLoadMu sync.Map // map[cacheKey]*sync.Mutex
)

func (p *ProcessorImpl) GetById(mapId _map.Id) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	key := cacheKey{tenantId: t.Id(), mapId: mapId}

	if cached, ok := mapCache.Load(key); ok {
		return cached.(Model), nil
	}

	muIface, _ := mapLoadMu.LoadOrStore(key, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	if cached, ok := mapCache.Load(key); ok {
		return cached.(Model), nil
	}

	m, err := requests.Provider[RestModel, Model](p.l, p.ctx)(requestMap(mapId), Extract)()
	if err != nil {
		return Model{}, err
	}
	mapCache.Store(key, m)
	return m, nil
}

// SnapMobPosition clamps a mob's (x, y) so y sits at most 1 px above the
// foothold's surface for the given fh. Centralized snap site for every
// outbound wire packet that carries a mob position (movement Kafka command,
// Spawn packet writer, Control packet writer).
//
// The v83 client validates positions in spawn packets against the foothold
// surface and treats at-or-below positions as embedded-in-terrain, dropping
// the mob through the floor. Mirrors Cosmic's MapleMap.addMonsterSpawn
// `newpos.y -= 1` invariant — every mob's y stays 1 px above its foothold.
//
// Pass-through (no clamp) cases:
//   - fh == 0 (mid-air, fall sequence): leaving y untouched is correct because
//     the mob legitimately has no resting foothold at this moment.
//   - foothold not found in map's tree: the client also won't be able to
//     validate against an unknown fh, so falling through to "accept as-is"
//     here matches what the client will do.
//   - x outside the foothold's horizontal span: same reasoning.
//   - GetById fails: log at Debug and pass through; do not break movement.
func SnapMobPosition(l logrus.FieldLogger, ctx context.Context, mapId _map.Id, x, y, fh int16) (int16, int16) {
	if fh == 0 {
		return x, y
	}
	m, err := NewProcessor(l, ctx).GetById(mapId)
	if err != nil {
		l.WithError(err).Debugf("Unable to load map [%d] for foothold snap; passing through (x=%d, y=%d, fh=%d).", mapId, x, y, fh)
		return x, y
	}
	surfaceY, ok := m.SurfaceYOnFoothold(uint32(fh), x)
	if !ok {
		return x, y
	}
	target := surfaceY - 1
	if y > target {
		return x, target
	}
	return x, y
}
