package party

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetByMemberId(characterId character.Id) (Model, error)
	GetById(partyId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(partyId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(partyId), Extract)()
}

func (p *ProcessorImpl) GetByMemberId(characterId character.Id) (Model, error) {
	rp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(characterId), Extract, model.Filters[Model]())
	return model.FirstProvider(rp, model.Filters[Model]())()
}
