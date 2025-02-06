package portal

import (
	"atlas-portals/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	portalsInMap  = "data/maps/%d/portals"
	portalsByName = portalsInMap + "?name=%s"
	portalById    = portalsInMap + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestInMapByName(mapId uint32, name string) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+portalsByName, mapId, name))
}

func requestInMapById(mapId uint32, id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+portalById, mapId, id))
}
