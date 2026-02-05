package storage

import (
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// GetStorage retrieves storage from atlas-storage
func GetStorage(l logrus.FieldLogger) func(ctx context.Context) func(accountId uint32, worldId world.Id) (RestModel, error) {
	return func(ctx context.Context) func(accountId uint32, worldId world.Id) (RestModel, error) {
		return func(accountId uint32, worldId world.Id) (RestModel, error) {
			return requestStorage(accountId, worldId)(l, ctx)
		}
	}
}

// GetAssets retrieves all assets from storage
func GetAssets(l logrus.FieldLogger) func(ctx context.Context) func(accountId uint32, worldId world.Id) ([]AssetRestModel, error) {
	return func(ctx context.Context) func(accountId uint32, worldId world.Id) ([]AssetRestModel, error) {
		return func(accountId uint32, worldId world.Id) ([]AssetRestModel, error) {
			return requestAssets(accountId, worldId)(l, ctx)
		}
	}
}
