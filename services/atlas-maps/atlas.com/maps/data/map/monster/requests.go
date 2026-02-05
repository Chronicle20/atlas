package monster

import (
	"atlas-maps/rest"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapsResource     = "data/maps/"
	monstersResource = mapsResource + "%d/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestSpawnPoints(mapId _map.Id) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, mapId))
}
