package equipable

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ByEquipmentIdModelProvider(equipmentId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(equipmentId), Extract)
}

func (p *Processor) GetById(equipmentId uint32) (Model, error) {
	return p.ByEquipmentIdModelProvider(equipmentId)()
}

func (p *Processor) Delete(equipmentId uint32) error {
	return deleteById(equipmentId)(p.l, p.ctx)
}

func (p *Processor) Create(itemId uint32) model.Provider[Model] {
	ro, err := requestCreate(itemId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to generate equipable information.")
		return model.ErrorProvider[Model](err)
	}
	return model.Map(Extract)(model.FixedProvider(ro))
}
