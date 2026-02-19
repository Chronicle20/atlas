package cash

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	cashItemResource = "data/cash/items/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(itemId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+cashItemResource, itemId))
}
