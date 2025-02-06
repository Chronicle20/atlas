package channel

import (
	"atlas-channel/rest"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	ChannelsResource = "worlds/%d/channels"
	ChannelResource  = ChannelsResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHANNELS")
}

func requestChannel(worldId byte, channelId byte) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))
}

func registerChannel(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, ipAddress string, port int) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, ipAddress string, port int) error {
		return func(worldId byte, channelId byte, ipAddress string, port int) error {
			i := RestModel{
				Id:        uint32(channelId),
				IpAddress: ipAddress,
				Port:      port,
			}
			_, err := rest.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+ChannelsResource, worldId), i)(l, ctx)
			return err
		}
	}
}

func unregisterChannel(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte) error {
	return func(ctx context.Context) func(worldId byte, channelId byte) error {
		return func(worldId byte, channelId byte) error {
			return rest.MakeDeleteRequest(fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))(l, ctx)
		}
	}
}
