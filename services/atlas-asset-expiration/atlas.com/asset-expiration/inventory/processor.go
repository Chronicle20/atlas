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
