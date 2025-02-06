package channel

import (
	"atlas-login/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	WorldsResource  = "worlds/"
	WorldResource   = WorldsResource + "%d"
	ForWorld        = WorldResource + "/channels"
	ChannelResource = WorldsResource + "%d/channels/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHANNELS")
}

func requestChannel(worldId byte, channelId byte) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))
}

func requestChannels(worldId byte) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ForWorld, worldId))
}
