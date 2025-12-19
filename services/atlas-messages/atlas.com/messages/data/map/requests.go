package _map

import (
	"atlas-messages/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	getMap = "data/maps/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestMap(mapId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+getMap, mapId))
}
