package _map

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
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

func (p *ProcessorImpl) CharacterIdsInFieldProvider(f field.Model) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInField(f), Extract, model.Filters[uint32]())
}
