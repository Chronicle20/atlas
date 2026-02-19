package storage

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	storageResource         = "storage/accounts/%d?worldId=%d"
	storageAssetsResource   = "storage/accounts/%d/assets?worldId=%d"
	projectionResource      = "storage/projections/%d"
	projectionAssetResource = "storage/projections/%d/compartments/%d/assets/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("STORAGE")
}

func requestStorageByAccountAndWorld(accountId uint32, worldId world.Id) requests.Request[StorageRestModel] {
	return requests.GetRequest[StorageRestModel](fmt.Sprintf(getBaseRequest()+storageResource, accountId, worldId))
}

func requestAssetsByAccountAndWorld(accountId uint32, worldId world.Id) requests.Request[[]AssetRestModel] {
	return requests.GetRequest[[]AssetRestModel](fmt.Sprintf(getBaseRequest()+storageAssetsResource, accountId, worldId))
}

func requestProjectionByCharacterId(characterId uint32) requests.Request[ProjectionRestModel] {
	return requests.GetRequest[ProjectionRestModel](fmt.Sprintf(getBaseRequest()+projectionResource, characterId))
}

func requestProjectionAsset(characterId uint32, compartmentType byte, slot int16) requests.Request[AssetRestModel] {
	return requests.GetRequest[AssetRestModel](fmt.Sprintf(getBaseRequest()+projectionAssetResource, characterId, compartmentType, slot))
}
