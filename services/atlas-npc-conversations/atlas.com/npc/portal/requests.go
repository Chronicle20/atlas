package portal

import (
	"atlas-npc-conversations/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	portalsInMap  = "maps/%d/portals"
	portalsByName = portalsInMap + "?name=%s"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestInMapByName(mapId uint32, name string) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+portalsByName, mapId, name))
}

func requestAll(mapId uint32) requests.Request[[]RestModel] {
	return requests.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+portalsInMap, mapId))
}
