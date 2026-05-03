package map_

import (
	"context"
	"sync"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

// Map data is static once loaded (foothold tree, return map, etc. don't
// change at runtime), so we cache forever process-wide. The cache is keyed
// by mapId; tenant differences in map data (if any) would need keying by
// tenant too, but no current data path differentiates per tenant.
var (
	mapCache   sync.Map // map[_map.Id]Model
	mapLoadMu  sync.Map // map[_map.Id]*sync.Mutex
)

func (p *ProcessorImpl) GetById(mapId _map.Id) (Model, error) {
	if cached, ok := mapCache.Load(mapId); ok {
		return cached.(Model), nil
	}

	muIface, _ := mapLoadMu.LoadOrStore(mapId, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	if cached, ok := mapCache.Load(mapId); ok {
		return cached.(Model), nil
	}

	m, err := requests.Provider[RestModel, Model](p.l, p.ctx)(requestMap(mapId), Extract)()
	if err != nil {
		return Model{}, err
	}
	mapCache.Store(mapId, m)
	return m, nil
}
