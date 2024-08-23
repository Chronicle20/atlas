package channel

import (
	"atlas-channel/rest"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
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

func requestChannel(ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte) requests.Request[RestModel] {
	return func(worldId byte, channelId byte) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))
	}
}

func registerChannel(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, ipAddress string, port int) error {
	return func(worldId byte, channelId byte, ipAddress string, port int) error {
		i := RestModel{
			Id:        uint32(channelId),
			IpAddress: ipAddress,
			Port:      port,
		}
		_, err := rest.MakePostRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+ChannelsResource, worldId), i)(l)
		return err
	}
}

func unregisterChannel(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte) error {
	return func(worldId byte, channelId byte) error {
		return rest.MakeDeleteRequest(ctx, tenant)(fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))(l)
	}
}
