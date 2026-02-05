package _map

import (
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
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

func (p *ProcessorImpl) CharacterIdsInMapProvider(field field.Model) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInMap(field), Extract, model.Filters[uint32]())
}
