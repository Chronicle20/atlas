package storage

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "storage/accounts/%d?worldId=%d"
	Assets   = "storage/accounts/%d/assets?worldId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("STORAGE")
}

func requestStorage(accountId uint32, worldId world.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId, worldId))
}

func requestAssets(accountId uint32, worldId world.Id) requests.Request[[]AssetRestModel] {
	return requests.GetRequest[[]AssetRestModel](fmt.Sprintf(getBaseRequest()+Assets, accountId, worldId))
}
