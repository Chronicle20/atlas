package object

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrUnknownObject is returned by GetDefaultState when the map declares no
// named object matching the requested name. Object names are opaque to the
// server, so this is an expected outcome, not an infrastructure failure.
var ErrUnknownObject = errors.New("map declares no object with that name")

type Processor interface {
	GetDefaultState(mapId _map.Id, name string) (uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetDefaultState resolves a named object's declared default state from
// atlas-data. It returns ErrUnknownObject when the map declares no such
// object.
func (p *ProcessorImpl) GetDefaultState(mapId _map.Id, name string) (uint32, error) {
	ms, err := p.inMapProvider(mapId)()
	if err != nil {
		return 0, err
	}
	for _, m := range ms {
		if m.Name() == name {
			return m.State(), nil
		}
	}
	return 0, ErrUnknownObject
}

// inMapProvider fetches every named object a map declares. atlas-data
// paginates its map collections, so this drains every page rather than
// fetching one.
func (p *ProcessorImpl) inMapProvider(mapId _map.Id) model.Provider[[]Model] {
	url, err := objectsUrl(p.ctx, mapId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
}
