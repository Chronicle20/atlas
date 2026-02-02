package inventory

import (
	"context"

	"github.com/sirupsen/logrus"
)

// GetInventory retrieves a character's inventory from atlas-inventory
func GetInventory(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (RestModel, error) {
	return func(ctx context.Context) func(characterId uint32) (RestModel, error) {
		return func(characterId uint32) (RestModel, error) {
			return requestInventory(characterId)(l, ctx)
		}
	}
}

// GetAssets retrieves assets from a specific compartment
func GetAssets(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, compartmentId string) ([]AssetRestModel, error) {
	return func(ctx context.Context) func(characterId uint32, compartmentId string) ([]AssetRestModel, error) {
		return func(characterId uint32, compartmentId string) ([]AssetRestModel, error) {
			return requestAssets(characterId, compartmentId)(l, ctx)
		}
	}
}

// GetEquippedAssets retrieves all equipped assets for a character (items in equipment slots)
func GetEquippedAssets(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]AssetRestModel, error) {
	return func(ctx context.Context) func(characterId uint32) ([]AssetRestModel, error) {
		return func(characterId uint32) ([]AssetRestModel, error) {
			inv, err := GetInventory(l)(ctx)(characterId)
			if err != nil {
				return nil, err
			}

			var equipped []AssetRestModel

			// Find the equip compartment and get equipped items
			for _, comp := range inv.Compartments {
				if comp.Type == "equip" || comp.Type == "EQUIP" {
					assets, err := GetAssets(l)(ctx)(characterId, comp.Id)
					if err != nil {
						l.WithError(err).Warnf("Failed to get assets for compartment [%s].", comp.Id)
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
	}
}

// GetCashAssets retrieves all cash assets for a character
func GetCashAssets(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]AssetRestModel, error) {
	return func(ctx context.Context) func(characterId uint32) ([]AssetRestModel, error) {
		return func(characterId uint32) ([]AssetRestModel, error) {
			inv, err := GetInventory(l)(ctx)(characterId)
			if err != nil {
				return nil, err
			}

			var cashAssets []AssetRestModel

			// Find the cash compartment
			for _, comp := range inv.Compartments {
				if comp.Type == "cash" || comp.Type == "CASH" {
					assets, err := GetAssets(l)(ctx)(characterId, comp.Id)
					if err != nil {
						l.WithError(err).Warnf("Failed to get assets for compartment [%s].", comp.Id)
						continue
					}

					cashAssets = append(cashAssets, assets...)
				}
			}

			return cashAssets, nil
		}
	}
}
