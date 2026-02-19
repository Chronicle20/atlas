package channel

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
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

func requestChannelsForWorld(worldId world.Id) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, worldId))
}

func requestChannel(ch channel.Model) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, ch.WorldId(), ch.Id()))
}
