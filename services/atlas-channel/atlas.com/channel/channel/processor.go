package channel

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"strconv"
)

func Register(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, ipAddress string, port string) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, ipAddress string, port string) error {
		return func(worldId byte, channelId byte, ipAddress string, portStr string) error {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				l.WithError(err).Errorf("Port [%s] is not a valid number.", portStr)
				return err
			}
			return registerChannel(l)(ctx)(worldId, channelId, ipAddress, port)
		}
	}
}

func Unregister(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte) error {
	return func(ctx context.Context) func(worldId byte, channelId byte) error {
		return func(worldId byte, channelId byte) error {
			return unregisterChannel(l)(ctx)(worldId, channelId)
		}
	}
}

func ByIdModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte) model.Provider[Model] {
	return func(ctx context.Context) func(worldId byte, channelId byte) model.Provider[Model] {
		return func(worldId byte, channelId byte) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestChannel(worldId, channelId), Extract)
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte) (Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte) (Model, error) {
		return func(worldId byte, channelId byte) (Model, error) {
			return ByIdModelProvider(l)(ctx)(worldId, channelId)()
		}
	}
}
