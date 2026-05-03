package info

import (
	"context"
	"sync"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	return &ProcessorImpl{l: l, ctx: ctx}
}

type cacheKey struct {
	tenantId uuid.UUID
	mapId    _map.Id
}

var (
	mapCache  sync.Map
	mapLoadMu sync.Map
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
