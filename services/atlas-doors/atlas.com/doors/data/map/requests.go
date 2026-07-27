package map_

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	getMap        = "data/maps/%d"
	getMapPortals = "data/maps/%d/portals"
)

var baseURLProvider = func() string {
	return requests.RootUrl("DATA")
}

func getBaseRequest() string {
	return baseURLProvider()
}

// requestMap fetches a map with portals included via ?include=portals.
func requestMap(mapId _map.Id) requests.Request[RestModel] {
	url := fmt.Sprintf(getBaseRequest()+getMap+"?include=portals", mapId)
	return requests.GetRequest[RestModel](url)
}

// portalsUrl is a bare URL (not a requests.Request) for the /portals
// sub-resource: the list is now paginated server-side (task-117) and
// consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func portalsUrl(mapId _map.Id) string {
	return fmt.Sprintf(getBaseRequest()+getMapPortals, mapId)
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrl("DATA") default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
