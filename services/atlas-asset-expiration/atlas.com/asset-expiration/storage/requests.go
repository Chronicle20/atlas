package storage

import (
	"atlas-asset-expiration/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "storage/accounts/%d?worldId=%d"
	Assets   = "storage/accounts/%d/assets?worldId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("STORAGE")
}

func requestStorage(accountId uint32, worldId byte) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId, worldId))
}

func requestAssets(accountId uint32, worldId byte) requests.Request[[]AssetRestModel] {
	return rest.MakeGetRequest[[]AssetRestModel](fmt.Sprintf(getBaseRequest()+Assets, accountId, worldId))
}
