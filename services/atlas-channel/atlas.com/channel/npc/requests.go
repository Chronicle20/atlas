package npc

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	npcsInMap           = "maps/%d/npcs"
	npcsInMapByObjectId = npcsInMap + "?objectId=%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestNPCsInMap(mapId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+npcsInMap, mapId))
}

func requestNPCsInMapByObjectId(mapId uint32, objectId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+npcsInMapByObjectId, mapId, objectId))
}
