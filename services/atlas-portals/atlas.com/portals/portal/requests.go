package portal

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	portalsInMap  = "data/maps/%d/portals"
	portalsByName = portalsInMap + "?name=%s"
	portalById    = portalsInMap + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// inMapUrl is a bare URL (not a requests.Request) because the list is now
// paginated server-side (task-117) and consumed via requests.DrainProvider,
// which appends its own page[number]/page[size] query params per request.
func inMapUrl(mapId _map.Id) string {
	return fmt.Sprintf(getBaseRequest()+portalsInMap, mapId)
}

func requestInMapByName(mapId _map.Id, name string) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+portalsByName, mapId, name))
}

func requestInMapById(mapId _map.Id, id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+portalById, mapId, id))
}
