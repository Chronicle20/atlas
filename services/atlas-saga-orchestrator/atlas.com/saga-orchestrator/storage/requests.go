package storage

import (
	"context"
	"fmt"

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

// RequestProjectionAsset retrieves a specific asset from a storage projection
func RequestProjectionAsset(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, compartmentType byte, slot int16) (ProjectionAssetRestModel, error) {
	return func(characterId uint32, compartmentType byte, slot int16) (ProjectionAssetRestModel, error) {
		url := fmt.Sprintf(getBaseRequest()+projectionAssetResource, characterId, compartmentType, slot)
		return requests.GetRequest[ProjectionAssetRestModel](url)(l, ctx)
	}
}
