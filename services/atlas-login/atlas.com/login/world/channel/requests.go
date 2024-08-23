package channel

import (
	"atlas-login/rest"
	"atlas-login/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	WorldsResource  = "worlds/"
	WorldResource   = WorldsResource + "%d"
	ForWorld        = WorldResource + "/channels"
	ChannelResource = WorldsResource + "%d/channels/%d"
)

func getBaseRequest() string {
	return os.Getenv("WORLD_SERVICE_URL")
}

func requestChannel(ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte) requests.Request[RestModel] {
	return func(worldId byte, channelId byte) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))
	}
}

func requestChannels(ctx context.Context, tenant tenant.Model) func(worldId byte) requests.Request[[]RestModel] {
	return func(worldId byte) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+ForWorld, worldId))
	}
}
