package _map

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	CharacterIdsInFieldProvider(f field.Model) model.Provider[[]uint32]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// CharacterIdsInFieldProvider fetches every character currently in one map
// instance, used for monster AI targeting/aggro. The upstream atlas-maps
// list is now paginated (task-117), so this drains every page rather than
// fetching just the first -- a truncated list here means monsters silently
// can't see (and can't aggro/attack) players beyond the first page.
func (p *ProcessorImpl) CharacterIdsInFieldProvider(f field.Model) model.Provider[[]uint32] {
	return requests.DrainProvider[RestModel, uint32](p.l, p.ctx)(charactersInFieldUrl(f), 250, Extract, model.Filters[uint32]())
}
