package portal

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	portalsInMap = "data/maps/%d/portals"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// allUrl is a bare URL (not a requests.Request) because the list is now
// paginated server-side (task-117) and consumed via requests.DrainProvider,
// which appends its own page[number]/page[size] query params per request.
func allUrl(mapId _map.Id) string {
	return fmt.Sprintf(getBaseRequest()+portalsInMap, mapId)
}
