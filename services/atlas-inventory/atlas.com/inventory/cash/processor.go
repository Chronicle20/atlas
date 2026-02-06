package cash

import (
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
	p.GetById = model.CollapseProvider(p.ByEquipmentIdModelProvider)
	return p
}

func (p *Processor) ByEquipmentIdModelProvider(id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(id), Extract)
}

func (p *Processor) UpdateQuantity(itemId uint32, quantity uint32) error {
	//TODO
	return nil
}

func (p *Processor) Delete(itemId uint32) error {
	return requestDelete(p.l, p.ctx)(itemId)
}

func (p *Processor) Create(templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
	return requestCreate(p.l, p.ctx)(templateId, commodityId, quantity, purchasedBy)
}
