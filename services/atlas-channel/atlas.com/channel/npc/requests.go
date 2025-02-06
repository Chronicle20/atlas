package npc

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	npcsInMap           = "maps/%d/npcs"
	npcsInMapByObjectId = npcsInMap + "?objectId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestNPCsInMap(mapId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+npcsInMap, mapId))
}

func requestNPCsInMapByObjectId(mapId uint32, objectId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+npcsInMapByObjectId, mapId, objectId))
}
