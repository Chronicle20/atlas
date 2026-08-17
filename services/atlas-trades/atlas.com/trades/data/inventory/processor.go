package inventory

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// ErrAssetNotFound is returned by AssetInSlot when the compartment read
// succeeded but the slot is empty. It is distinct from a transport failure on
// purpose: an empty slot means the client addressed an item it does not have,
// while a transport failure means atlas-inventory is unreachable.
var ErrAssetNotFound = errors.New("inventory: no asset in slot")

// Processor is the inventory REST client the staging path reads through.
type Processor interface {
	CompartmentProvider(characterId character.Id, inventoryType inventory.Type) model.Provider[Model]
	GetCompartment(characterId character.Id, inventoryType inventory.Type) (Model, error)
	// AssetInSlot returns the asset occupying (inventoryType, s) for the
	// character, or ErrAssetNotFound when the slot is empty.
	AssetInSlot(characterId character.Id, inventoryType inventory.Type, s slot.Position) (Asset, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) CompartmentProvider(characterId character.Id, inventoryType inventory.Type) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByType(p.ctx, characterId, inventoryType), Extract)
}

func (p *ProcessorImpl) GetCompartment(characterId character.Id, inventoryType inventory.Type) (Model, error) {
	return p.CompartmentProvider(characterId, inventoryType)()
}

func (p *ProcessorImpl) AssetInSlot(characterId character.Id, inventoryType inventory.Type, s slot.Position) (Asset, error) {
	c, err := p.GetCompartment(characterId, inventoryType)
	if err != nil {
		return Asset{}, err
	}
	a, ok := c.FindBySlot(s)
	if !ok {
		return Asset{}, ErrAssetNotFound
	}
	return a, nil
}
