package portal

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	portalsInMap  = "data/maps/%d/portals"
	portalsByName = portalsInMap + "?name=%s"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestInMapByName(mapId uint32, name string) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+portalsByName, mapId, name))
}
