package inventory

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetInventory(characterId uint32) (RestModel, error)
	GetAssets(characterId uint32, compartmentId string) ([]AssetRestModel, error)
	GetEquippedAssets(characterId uint32) ([]AssetRestModel, error)
	GetCashAssets(characterId uint32) ([]AssetRestModel, error)
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

// GetInventory retrieves a character's inventory from atlas-inventory
func (p *ProcessorImpl) GetInventory(characterId uint32) (RestModel, error) {
	return requestInventory(characterId)(p.l, p.ctx)
}

// GetAssets retrieves assets from a specific compartment
func (p *ProcessorImpl) GetAssets(characterId uint32, compartmentId string) ([]AssetRestModel, error) {
	return requestAssets(characterId, compartmentId)(p.l, p.ctx)
}

// GetEquippedAssets retrieves all equipped assets for a character (items in equipment slots)
func (p *ProcessorImpl) GetEquippedAssets(characterId uint32) ([]AssetRestModel, error) {
	inv, err := p.GetInventory(characterId)
	if err != nil {
		return nil, err
	}

	var equipped []AssetRestModel

	// Find the equip compartment and get equipped items
	for _, comp := range inv.Compartments {
		if comp.Type == CompartmentTypeEquip {
			assets, err := p.GetAssets(characterId, comp.Id)
			if err != nil {
				p.l.WithError(err).Warnf("Failed to get assets for compartment [%s].", comp.Id)
				continue
			}

			// Filter to only equipped items (negative slot)
			for _, asset := range assets {
				if asset.IsEquipmentSlot() && asset.IsEquipable() {
					equipped = append(equipped, asset)
				}
			}
		}
	}

	return equipped, nil
}

// GetCashAssets retrieves all cash assets for a character
func (p *ProcessorImpl) GetCashAssets(characterId uint32) ([]AssetRestModel, error) {
	inv, err := p.GetInventory(characterId)
	if err != nil {
		return nil, err
	}

	var cashAssets []AssetRestModel

	// Find the cash compartment
	for _, comp := range inv.Compartments {
		if comp.Type == CompartmentTypeCash {
			assets, err := p.GetAssets(characterId, comp.Id)
			if err != nil {
				p.l.WithError(err).Warnf("Failed to get assets for compartment [%s].", comp.Id)
				continue
			}

			cashAssets = append(cashAssets, assets...)
		}
	}

	return cashAssets, nil
}
