package npc

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	npcsInMap           = "data/maps/%d/npcs"
	npcsInMapByObjectId = npcsInMap + "?objectId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// npcsInMapUrl is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func npcsInMapUrl(mapId _map.Id) string {
	return fmt.Sprintf(getBaseRequest()+npcsInMap, mapId)
}

func requestNPCsInMapByObjectId(mapId _map.Id, objectId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+npcsInMapByObjectId, mapId, objectId))
}
