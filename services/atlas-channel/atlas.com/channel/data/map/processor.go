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
