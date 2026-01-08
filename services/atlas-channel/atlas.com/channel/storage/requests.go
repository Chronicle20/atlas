package storage

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	storageResource       = "storage/accounts/%d?worldId=%d"
	storageAssetsResource = "storage/accounts/%d/assets?worldId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("STORAGE")
}

func requestStorageByAccountAndWorld(accountId uint32, worldId byte) requests.Request[StorageRestModel] {
	return rest.MakeGetRequest[StorageRestModel](fmt.Sprintf(getBaseRequest()+storageResource, accountId, worldId))
}

func requestAssetsByAccountAndWorld(accountId uint32, worldId byte) requests.Request[[]AssetRestModel] {
	return rest.MakeGetRequest[[]AssetRestModel](fmt.Sprintf(getBaseRequest()+storageAssetsResource, accountId, worldId))
}
