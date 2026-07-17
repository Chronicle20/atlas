package map_

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(mapId _map.Id) (Model, error)
	GetPortals(mapId _map.Id) ([]Portal, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetById fetches the map from atlas-data including its portals via the
// ?include=portals query parameter, so a single round-trip populates both map
// attributes and portal sub-resources.
func (p *ProcessorImpl) GetById(mapId _map.Id) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestMap(mapId), Extract)()
}

// GetPortals fetches only the portal list for a map via the /portals
// sub-resource endpoint.
func (p *ProcessorImpl) GetPortals(mapId _map.Id) ([]Portal, error) {
	return requests.SliceProvider[PortalRestModel, Portal](p.l, p.ctx)(requestPortals(mapId), ExtractPortal, model.Filters[Portal]())()
}
