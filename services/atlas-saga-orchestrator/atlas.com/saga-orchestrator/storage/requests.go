package storage

import (
	"atlas-saga-orchestrator/rest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	storageAssetsResource   = "storage/accounts/%d/assets?worldId=%d"
	projectionAssetResource = "storage/projections/%d/compartments/%d/assets/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("STORAGE")
}

// RequestAssets retrieves all assets from storage for an account and world
func RequestAssets(l logrus.FieldLogger, ctx context.Context) func(accountId uint32, worldId world.Id) ([]AssetRestModel, error) {
	return func(accountId uint32, worldId world.Id) ([]AssetRestModel, error) {
		url := fmt.Sprintf(getBaseRequest()+storageAssetsResource, accountId, worldId)
		return rest.MakeGetRequest[[]AssetRestModel](url)(l, ctx)
	}
}

// RequestProjectionAsset retrieves a specific asset from a storage projection
func RequestProjectionAsset(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, compartmentType byte, slot int16) (ProjectionAssetRestModel, error) {
	return func(characterId uint32, compartmentType byte, slot int16) (ProjectionAssetRestModel, error) {
		url := fmt.Sprintf(getBaseRequest()+projectionAssetResource, characterId, compartmentType, slot)
		return rest.MakeGetRequest[ProjectionAssetRestModel](url)(l, ctx)
	}
}
