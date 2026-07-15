package world

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	WorldsResource        = "worlds"
	WorldsIncludeChannels = WorldsResource + "?include=channels"
	WorldsById            = WorldsResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("WORLDS")
}

// worldsUrl is a bare URL (not a requests.Request) because the list is now
// paginated server-side (task-117) and consumed via requests.DrainProvider,
// which appends its own page[number]/page[size] query params per request.
func worldsUrl() string {
	return getBaseRequest() + WorldsIncludeChannels
}

func requestWorld(worldId world.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+WorldsById, worldId))
}
