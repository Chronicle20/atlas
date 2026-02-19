package script

import (
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapsResource = "data/maps/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestMapScripts(mapId _map.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapsResource, mapId))
}
