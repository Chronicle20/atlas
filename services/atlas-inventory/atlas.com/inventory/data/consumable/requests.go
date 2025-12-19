package consumable

import (
	"atlas-inventory/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource     = "data/consumables"
	ById         = Resource + "/%d"
	Rechargeable = Resource + "?fields[consumables]=rechargeable&filter[rechargeable]=true"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestRechargeable() requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](getBaseRequest() + Rechargeable)
}
