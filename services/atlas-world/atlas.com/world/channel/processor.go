package channel

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func allProvider() model.SliceProvider[Model] {
	return func() ([]Model, error) {
		return GetChannelRegistry().ChannelServers(), nil
	}
}

func byWorldProvider(_ logrus.FieldLogger) func(worldId byte) model.SliceProvider[Model] {
	return func(worldId byte) model.SliceProvider[Model] {
		return model.FilteredProvider[Model](allProvider(), ByWorldFilter(worldId))
	}
}

func ByWorldFilter(id byte) model.Filter[Model] {
	return func(m Model) bool {
		return m.worldId == id
	}
}

func GetByWorld(l logrus.FieldLogger) func(worldId byte) ([]Model, error) {
	return func(worldId byte) ([]Model, error) {
		return byWorldProvider(l)(worldId)()
	}
}

func byIdProvider(_ logrus.FieldLogger) func(worldId byte, channelId byte) model.Provider[Model] {
	return func(worldId byte, channelId byte) model.Provider[Model] {
		return func() (Model, error) {
			return GetChannelRegistry().ChannelServer(worldId, channelId)
		}
	}
}

func GetById(l logrus.FieldLogger) func(worldId byte, channelId byte) (Model, error) {
	return func(worldId byte, channelId byte) (Model, error) {
		return byIdProvider(l)(worldId, channelId)()
	}
}

func Register(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
	return GetChannelRegistry().Register(worldId, channelId, ipAddress, port), nil
}
