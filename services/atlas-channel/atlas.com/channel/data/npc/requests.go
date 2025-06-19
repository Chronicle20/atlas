package npc

import (
	"atlas-channel/rest"
	"fmt"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	npcsInMap           = "data/maps/%d/npcs"
	npcsInMapByObjectId = npcsInMap + "?objectId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestNPCsInMap(mapId _map.Id) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+npcsInMap, mapId))
}

func requestNPCsInMapByObjectId(mapId _map.Id, objectId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+npcsInMapByObjectId, mapId, objectId))
}
