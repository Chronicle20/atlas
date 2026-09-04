package compartment

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	ByCharacterIdAndTypeProvider(characterId uint32, inventoryType inventory.Type) model.Provider[Model]
	// GetByType reads one compartment snapshot. atlas-inventory has no
	// batched all-compartments endpoint and requires the type query param
	// (services/atlas-inventory/atlas.com/inventory/compartment/resource.go);
	// a full snapshot is therefore three calls (EQUIP, USE, ETC), not one.
	GetByType(characterId uint32, inventoryType inventory.Type) (Model, error)
	// CanAccommodate asks atlas-inventory whether it would currently accept a
	// grant of every item in items to characterId — merge-aware, so a full
	// compartment does not block a stackable that fits an existing stack. Do
	// not re-derive this rule locally; atlas-inventory owns it
	// (compartment.CanAccommodate/accommodatesOne).
	CanAccommodate(characterId uint32, items []AccommodationItem) (bool, error)
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

func (p *ProcessorImpl) ByCharacterIdAndTypeProvider(characterId uint32, inventoryType inventory.Type) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByType(p.ctx, characterId, inventoryType), Extract)
}

func (p *ProcessorImpl) GetByType(characterId uint32, inventoryType inventory.Type) (Model, error) {
	return p.ByCharacterIdAndTypeProvider(characterId, inventoryType)()
}

func (p *ProcessorImpl) CanAccommodate(characterId uint32, items []AccommodationItem) (bool, error) {
	return requests.Provider[accommodationOutputRestModel, bool](p.l, p.ctx)(
		requestCheckAccommodation(p.ctx, characterId, items),
		func(rm accommodationOutputRestModel) (bool, error) { return rm.Accommodated, nil },
	)()
}
