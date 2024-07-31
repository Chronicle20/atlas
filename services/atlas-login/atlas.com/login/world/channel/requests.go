package channel

import (
	"atlas-login/rest"
	"atlas-login/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
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

func requestChannel(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte) requests.Request[RestModel] {
	return func(worldId byte, channelId byte) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))
	}
}

func requestChannels(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) requests.Request[[]RestModel] {
	return func(worldId byte) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+ForWorld, worldId))
	}
}
