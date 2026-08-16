package storage

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	storageResource         = "storage/accounts/%d?worldId=%d"
	storageAssetsResource   = "storage/accounts/%d/assets?worldId=%d"
	projectionResource      = "storage/projections/%d"
	projectionAssetResource = "storage/projections/%d/compartments/%d/assets/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "STORAGE")
}

func requestStorageByAccountAndWorld(ctx context.Context, accountId uint32, worldId world.Id) requests.Request[StorageRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[StorageRestModel](err)
	}
	return requests.GetRequest[StorageRestModel](fmt.Sprintf(root+storageResource, accountId, worldId))
}

func requestProjectionByCharacterId(ctx context.Context, characterId uint32) requests.Request[ProjectionRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[ProjectionRestModel](err)
	}
	return requests.GetRequest[ProjectionRestModel](fmt.Sprintf(root+projectionResource, characterId))
}
