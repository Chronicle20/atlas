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

func requestAssetsByAccountAndWorld(ctx context.Context, accountId uint32, worldId world.Id) requests.Request[[]AssetRestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]AssetRestModel](err)
	}
	return requests.GetRequest[[]AssetRestModel](fmt.Sprintf(root+storageAssetsResource, accountId, worldId))
}

func requestProjectionByCharacterId(ctx context.Context, characterId uint32) requests.Request[ProjectionRestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[ProjectionRestModel](err)
	}
	return requests.GetRequest[ProjectionRestModel](fmt.Sprintf(root+projectionResource, characterId))
}

func requestProjectionAsset(ctx context.Context, characterId uint32, compartmentType byte, slot int16) requests.Request[AssetRestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[AssetRestModel](err)
	}
	return requests.GetRequest[AssetRestModel](fmt.Sprintf(root+projectionAssetResource, characterId, compartmentType, slot))
}
