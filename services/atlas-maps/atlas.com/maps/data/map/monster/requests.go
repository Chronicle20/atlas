package monster

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapsResource     = "data/maps/"
	monstersResource = mapsResource + "%d/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// spawnPointsUrl is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func spawnPointsUrl(mapId _map.Id) string {
	return fmt.Sprintf(getBaseRequest()+monstersResource, mapId)
}
