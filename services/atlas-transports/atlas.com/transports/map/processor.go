package _map

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	CharacterIdsInMapProvider(field field.Model) model.Provider[[]uint32]
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

// CharacterIdsInMapProvider fetches every character currently in one map
// instance, used to identify passengers to board/notify for transport
// departures. The upstream atlas-maps list is now paginated (task-117), so
// this drains every page rather than fetching just the first -- a truncated
// list here means some passengers silently miss the transport.
func (p *ProcessorImpl) CharacterIdsInMapProvider(field field.Model) model.Provider[[]uint32] {
	return requests.DrainProvider[RestModel, uint32](p.l, p.ctx)(charactersInMapUrl(field), 250, Extract, model.Filters[uint32]())
}
