package data

import (
	"atlas-messages/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	getMap = "maps/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestMap(mapId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+getMap, mapId))
}
