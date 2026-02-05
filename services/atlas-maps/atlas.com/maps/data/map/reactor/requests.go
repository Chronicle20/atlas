package reactor

import (
	"atlas-maps/rest"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapsResource     = "data/maps/"
	reactorsResource = mapsResource + "%d/reactors"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestReactors(mapId _map.Id) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+reactorsResource, mapId))
}
