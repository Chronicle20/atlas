package map_

import (
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	getMap = "data/maps/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestMap(mapId _map.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+getMap, mapId))
}
