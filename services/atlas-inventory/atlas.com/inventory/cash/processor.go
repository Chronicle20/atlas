package cash

import (
	model2 "atlas-inventory/model"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l       logrus.FieldLogger
	ctx     context.Context
	GetById func(itemId uint32) (Model, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	p.GetById = model2.CollapseProvider(p.ByEquipmentIdModelProvider)
	return p
}

func (p *Processor) ByEquipmentIdModelProvider(itemId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)
}

func (p *Processor) UpdateQuantity(itemId uint32, quantity uint32) error {
	//TODO
	return nil
}
