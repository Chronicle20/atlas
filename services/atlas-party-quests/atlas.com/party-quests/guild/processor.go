package guild

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetByMemberId(memberId uint32) (Model, error)
	ByMemberIdProvider(memberId uint32) model.Provider[[]Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) GetByMemberId(memberId uint32) (Model, error) {
	return model.First[Model](p.ByMemberIdProvider(memberId), model.Filters[Model]())
}

func (p *ProcessorImpl) ByMemberIdProvider(memberId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
}
