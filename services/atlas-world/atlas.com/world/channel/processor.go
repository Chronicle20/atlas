package channel

import (
	"atlas-world/configuration"
	"atlas-world/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func allProvider(tenant tenant.Model) model.SliceProvider[Model] {
	return func() ([]Model, error) {
		return GetChannelRegistry().ChannelServers(tenant.Id.String()), nil
	}
}

func byWorldProvider(_ logrus.FieldLogger, tenant tenant.Model) func(worldId byte) model.SliceProvider[Model] {
	return func(worldId byte) model.SliceProvider[Model] {
		return model.FilteredProvider[Model](allProvider(tenant), ByWorldFilter(worldId))
	}
}

func ByWorldFilter(id byte) model.Filter[Model] {
	return func(m Model) bool {
		return m.worldId == id
	}
}

func GetByWorld(l logrus.FieldLogger, tenant tenant.Model) func(worldId byte) ([]Model, error) {
	return func(worldId byte) ([]Model, error) {
		return byWorldProvider(l, tenant)(worldId)()
	}
}

func byIdProvider(_ logrus.FieldLogger, tenant tenant.Model) func(worldId byte, channelId byte) model.Provider[Model] {
	return func(worldId byte, channelId byte) model.Provider[Model] {
		return func() (Model, error) {
			return GetChannelRegistry().ChannelServer(tenant.Id.String(), worldId, channelId)
		}
	}
}

func GetById(l logrus.FieldLogger, tenant tenant.Model) func(worldId byte, channelId byte) (Model, error) {
	return func(worldId byte, channelId byte) (Model, error) {
		return byIdProvider(l, tenant)(worldId, channelId)()
	}
}

func Register(_ logrus.FieldLogger, tenant tenant.Model) func(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
	return func(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
		return GetChannelRegistry().Register(tenant.Id.String(), worldId, channelId, ipAddress, port), nil
	}
}

func Unregister(_ logrus.FieldLogger, tenant tenant.Model) func(worldId byte, channelId byte) error {
	return func(worldId byte, channelId byte) error {
		GetChannelRegistry().RemoveByWorldAndChannel(tenant.Id.String(), worldId, channelId)
		return nil
	}
}

func RequestStatus(l logrus.FieldLogger, span opentracing.Span, c configuration.Model) {
	for _, sc := range c.Data.Attributes.Servers {
		t := tenant.New(uuid.MustParse(sc.Tenant), "", 0, 0)
		emitChannelServerStatusCommand(l, span, t)
	}
}
