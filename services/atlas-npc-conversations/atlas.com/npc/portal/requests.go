package portal

import (
	"atlas-npc-conversations/rest"
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

func requestAll(mapId uint32) requests.Request[[]RestModel] {
	return requests.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+portalsInMap, mapId))
}
