package channel

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func Register(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id, ipAddress string, port int) error {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id, ipAddress string, port int) error {
		return func(worldId world.Id, channelId channel.Id, ipAddress string, port int) error {
			return registerChannel(l)(ctx)(worldId, channelId, ipAddress, port)
		}
	}
}

func Unregister(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id) error {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id) error {
		return func(worldId world.Id, channelId channel.Id) error {
			return unregisterChannel(l)(ctx)(worldId, channelId)
		}
	}
}

func ByIdModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id) model.Provider[Model] {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id) model.Provider[Model] {
		return func(worldId world.Id, channelId channel.Id) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestChannel(worldId, channelId), Extract)
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id) (Model, error) {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id) (Model, error) {
		return func(worldId world.Id, channelId channel.Id) (Model, error) {
			return ByIdModelProvider(l)(ctx)(worldId, channelId)()
		}
	}
}
