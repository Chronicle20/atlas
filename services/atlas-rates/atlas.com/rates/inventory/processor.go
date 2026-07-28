package inventory

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

// identity is the no-op transformer for requests.DrainProvider, since
// AssetRestModel is already the target type for this consumer.
func identity(m AssetRestModel) (AssetRestModel, error) {
	return m, nil
}

var _ Processor = (*ProcessorImpl)(nil)

// GetInventory retrieves a character's inventory from atlas-inventory
func (p *ProcessorImpl) GetInventory(characterId uint32) (RestModel, error) {
	return requestInventory(characterId)(p.l, p.ctx)
}

// GetAssets retrieves ALL assets from a specific compartment. The upstream
// list is now paginated server-side (task-117); GetEquippedAssets/GetCashAssets
// below need every asset in the compartment, so this drains every page
// rather than fetching just the first.
func (p *ProcessorImpl) GetAssets(characterId uint32, compartmentId string) ([]AssetRestModel, error) {
	return requests.DrainProvider[AssetRestModel, AssetRestModel](p.l, p.ctx)(compartmentAssetsUrl(characterId, compartmentId), 250, identity, model.Filters[AssetRestModel]())()
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
