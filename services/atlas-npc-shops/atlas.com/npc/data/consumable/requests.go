package consumable

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource     = "data/consumables"
	ById         = Resource + "/%d"
	Rechargeable = Resource + "?fields[consumables]=rechargeable,slotMax,unitPrice&filter[rechargeable]=true"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestRechargeable() requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](getBaseRequest() + Rechargeable)
}
