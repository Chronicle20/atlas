package info

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const mapsResource = "data/maps/%d"

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestMap(mapId _map.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapsResource, mapId))
}
