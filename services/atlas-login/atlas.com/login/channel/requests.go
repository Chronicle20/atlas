package channel

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	WorldsResource = "worlds/"
	WorldsById     = WorldsResource + "%d"
	Resource       = WorldsById + "/channels"
	ById           = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHANNELS")
}

// channelsForWorldUrl is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func channelsForWorldUrl(worldId world.Id) string {
	return fmt.Sprintf(getBaseRequest()+Resource, worldId)
}

func requestChannel(ch channel.Model) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, ch.WorldId(), ch.Id()))
}
