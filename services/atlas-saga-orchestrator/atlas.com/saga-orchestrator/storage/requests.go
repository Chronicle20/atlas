package storage

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	storageAssetsResource   = "storage/accounts/%d/assets?worldId=%d"
	projectionAssetResource = "storage/projections/%d/compartments/%d/assets/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "STORAGE")
}

// RequestProjectionAsset retrieves a specific asset from a storage projection
func RequestProjectionAsset(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, compartmentType byte, slot int16) (ProjectionAssetRestModel, error) {
	return func(characterId uint32, compartmentType byte, slot int16) (ProjectionAssetRestModel, error) {
		root, err := getBaseRequest(ctx)
		if err != nil {
			return ProjectionAssetRestModel{}, err
		}
		url := fmt.Sprintf(root+projectionAssetResource, characterId, compartmentType, slot)
		return requests.GetRequest[ProjectionAssetRestModel](url)(l, ctx)
	}
}
