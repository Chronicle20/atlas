package channel

import (
	"atlas-channel/rest"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
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

func requestChannel(worldId world.Id, channelId channel.Id) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ChannelResource, worldId, channelId))
}

func registerChannel(l logrus.FieldLogger) func(ctx context.Context) func(c Model) error {
	return func(ctx context.Context) func(c Model) error {
		return func(c Model) error {
			i, err := model.Map(Transform)(model.FixedProvider(c))()
			if err != nil {
				return err
			}
			_, err = rest.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+ChannelsResource, c.WorldId()), i)(l, ctx)
			return err
		}
	}
}
