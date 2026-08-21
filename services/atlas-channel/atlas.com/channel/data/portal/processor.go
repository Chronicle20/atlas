package portal

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	InMapByNameModelProvider(mapId _map.Id, name string) model.Provider[[]Model]
	GetInMapByName(mapId _map.Id, name string) (Model, error)
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

var _ Processor = (*ProcessorImpl)(nil)

// requestInMapFn is the REST fallback seam for getPortalsInMap. Package-level
// var so tests can point it at an httptest server without going through the
// tenant/service-URL context the libs/atlas-rest pipeline reads.
var requestInMapFn = requestInMap

// cacheKey scopes the per-map portal cache by tenant, mirroring
// data/map/processor.go's cacheKey.
type cacheKey struct {
	tenantId uuid.UUID
	mapId    _map.Id
}

// Portal data is static WZ data, cached for the process lifetime — pod
// restart is the invalidation contract.
var (
	portalCache  sync.Map // map[cacheKey][]Model
	portalLoadMu sync.Map // map[cacheKey]*sync.Mutex
)

func (p *ProcessorImpl) getPortalsInMap(mapId _map.Id) ([]Model, error) {
	t := tenant.MustFromContext(p.ctx)
	key := cacheKey{tenantId: t.Id(), mapId: mapId}

	if cached, ok := portalCache.Load(key); ok {
		return cached.([]Model), nil
	}

	muIface, _ := portalLoadMu.LoadOrStore(key, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	if cached, ok := portalCache.Load(key); ok {
		return cached.([]Model), nil
	}

	ms, err := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMapFn(p.ctx, mapId), Extract, model.Filters[Model]())()
	if err != nil {
		return nil, err
	}
	portalCache.Store(key, ms)
	return ms, nil
}

func (p *ProcessorImpl) InMapByNameModelProvider(mapId _map.Id, name string) model.Provider[[]Model] {
	ms, err := p.getPortalsInMap(mapId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	filtered := make([]Model, 0)
	for _, m := range ms {
		if m.Name() == name {
			filtered = append(filtered, m)
		}
	}
	return model.FixedProvider(filtered)
}

func (p *ProcessorImpl) GetInMapByName(mapId _map.Id, name string) (Model, error) {
	ms, err := p.getPortalsInMap(mapId)
	if err != nil {
		return Model{}, err
	}
	for _, m := range ms {
		if m.Name() == name {
			return m, nil
		}
	}
	return Model{}, fmt.Errorf("no portal named [%s] in map [%d]", name, mapId)
}
