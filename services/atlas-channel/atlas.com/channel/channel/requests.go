package channel

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	ChannelsResource = "worlds/%d/channels"
	ChannelResource  = ChannelsResource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHANNELS")
}

func requestChannel(ctx context.Context, ch channel.Model) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ChannelResource, ch.WorldId(), ch.Id()))
}

func unregisterChannel(ctx context.Context, ch channel.Model) requests.EmptyBodyRequest {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return func(l logrus.FieldLogger, _ context.Context) error { return err }
	}
	return requests.DeleteRequest(fmt.Sprintf(root+ChannelResource, ch.WorldId(), ch.Id()))
}

func registerChannel(l logrus.FieldLogger) func(ctx context.Context) func(c Model) error {
	return func(ctx context.Context) func(c Model) error {
		return func(c Model) error {
			i, err := model.Map(Transform)(model.FixedProvider(c))()
			if err != nil {
				return err
			}
			root, err := getBaseRequest(ctx)
			if err != nil {
				return err
			}
			_, err = requests.PostRequest[RestModel](fmt.Sprintf(root+ChannelsResource, c.WorldId()), i)(l, ctx)
			return err
		}
	}
}
