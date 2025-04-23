package portal

import (
	"atlas-messages/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	portalsInMap = "map/maps/%d/portals"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestAll(mapId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+portalsInMap, mapId))
}
