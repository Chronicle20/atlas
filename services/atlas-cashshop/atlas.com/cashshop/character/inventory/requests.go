package inventory

import (
	"atlas-cashshop/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters/%d/inventory"
	ById     = Resource
)

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
