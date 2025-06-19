package setup

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(id uint32) (Model, error)
	ByIdModelProvider(id uint32) model.Provider[Model]
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

func (p *ProcessorImpl) ByIdModelProvider(id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(id), Extract)
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return p.ByIdModelProvider(id)()
}

func Extract(m RestModel) (Model, error) {
	return Model{
		id:         m.Id,
		price:      m.Price,
		slotMax:    m.SlotMax,
		recoveryHP: m.RecoveryHP,
		tradeBlock: m.TradeBlock,
		notSale:    m.NotSale,
		reqLevel:   m.ReqLevel,
		distanceX:  m.DistanceX,
		distanceY:  m.DistanceY,
		maxDiff:    m.MaxDiff,
		direction:  m.Direction,
	}, nil
}
