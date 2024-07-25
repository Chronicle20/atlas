package channel

import (
	"atlas-channel/rest"
	"atlas-channel/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	ChannelsResource = "worlds/%d/channels"
	ChannelResource  = ChannelsResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("WORLD_SERVICE_URL")
}

func requestChannel(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte) requests.Request[RestModel] {
	return func(worldId byte, channelId byte) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))
	}
}

func registerChannel(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, ipAddress string, port int) error {
	return func(worldId byte, channelId byte, ipAddress string, port int) error {
		i := RestModel{
			Id:        uint32(channelId),
			IpAddress: ipAddress,
			Port:      port,
		}
		_, err := rest.MakePostRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+ChannelsResource, worldId), i)(l)
		return err
	}
}

func unregisterChannel(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte) error {
	return func(worldId byte, channelId byte) error {
		return rest.MakeDeleteRequest(l, span, tenant)(fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))(l)
	}
}
