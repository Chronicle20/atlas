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

// GetAllAssets retrieves all assets across all compartments for a character
func GetAllAssets(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]AssetRestModel, map[uint32]uint8, error) {
	return func(ctx context.Context) func(characterId uint32) ([]AssetRestModel, map[uint32]uint8, error) {
		return func(characterId uint32) ([]AssetRestModel, map[uint32]uint8, error) {
			inv, err := GetInventory(l)(ctx)(characterId)
			if err != nil {
				return nil, nil, err
			}

			var allAssets []AssetRestModel
			// Map asset ID (parsed from string) to compartment type
			assetCompartmentTypes := make(map[uint32]uint8)

			for _, comp := range inv.Compartments {
				assets, err := GetAssets(l)(ctx)(characterId, comp.Id)
				if err != nil {
					l.WithError(err).Warnf("Failed to get assets for compartment [%s].", comp.Id)
					continue
				}

				for _, asset := range assets {
					allAssets = append(allAssets, asset)
					// Store compartment type for each asset (using template ID as temporary key)
					// Note: We'll need the actual asset ID for commands
				}

				// Store compartment type for all assets in this compartment
				for i := range assets {
					// Parse asset ID and store compartment type
					assetCompartmentTypes[assets[i].TemplateId] = comp.Type
				}
			}

			return allAssets, assetCompartmentTypes, nil
		}
	}
}
