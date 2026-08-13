package chat

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	// RecentInvolving returns the buffered chat lines authored by any of the
	// listed characters, merged and sorted ascending by timestamp.
	RecentInvolving(characterIds []uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) RecentInvolving(characterIds []uint32) ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestHistory(characterIds), Extract, model.Filters[Model]())()
}
