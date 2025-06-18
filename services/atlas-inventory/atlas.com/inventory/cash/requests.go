package cash

import (
	"atlas-inventory/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	itemsResource = "cash-shop/items"
	itemResource  = itemsResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+itemResource, id))
}
