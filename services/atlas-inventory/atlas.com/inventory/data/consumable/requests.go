package consumable

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

// rechargeableUrl is a bare URL (not a requests.Request) because the
// filter[rechargeable]=true list is now paginated server-side (task-117)
// and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func rechargeableUrl() string {
	return getBaseRequest() + Rechargeable
}
