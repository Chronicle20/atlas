package compartment

import (
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByCharacterIdAndTypeProvider(characterId uint32, inventoryType inventory.Type) model.Provider[Model]
	GetByType(characterId uint32, inventoryType inventory.Type) (Model, error)
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

func (p *ProcessorImpl) ByCharacterIdAndTypeProvider(characterId uint32, inventoryType inventory.Type) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByType(characterId, inventoryType), Extract)
}

func (p *ProcessorImpl) GetByType(characterId uint32, inventoryType inventory.Type) (Model, error) {
	return p.ByCharacterIdAndTypeProvider(characterId, inventoryType)()
}
