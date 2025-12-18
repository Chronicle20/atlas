package reactor

import (
	"atlas-maps/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapsResource     = "data/maps/"
	reactorsResource = mapsResource + "%d/reactors"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestReactors(mapId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+reactorsResource, mapId))
}
