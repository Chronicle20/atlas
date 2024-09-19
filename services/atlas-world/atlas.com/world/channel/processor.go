package channel

import (
	"atlas-world/configuration"
	"atlas-world/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func AllProvider(ctx context.Context) model.Provider[[]Model] {
	return func() ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return GetChannelRegistry().ChannelServers(t.Id().String()), nil
	}
}

func ByWorldProvider(ctx context.Context) func(worldId byte) model.Provider[[]Model] {
	return func(worldId byte) model.Provider[[]Model] {
		return model.FilteredProvider[Model](AllProvider(ctx), model.Filters(ByWorldFilter(worldId)))
	}
}

func ByWorldFilter(id byte) model.Filter[Model] {
	return func(m Model) bool {
		return m.worldId == id
	}
}

func GetByWorld(_ logrus.FieldLogger) func(ctx context.Context) func(worldId byte) ([]Model, error) {
	return func(ctx context.Context) func(worldId byte) ([]Model, error) {
		return func(worldId byte) ([]Model, error) {
			return ByWorldProvider(ctx)(worldId)()
		}
	}
}

func ByIdProvider(ctx context.Context) func(worldId byte, channelId byte) model.Provider[Model] {
	return func(worldId byte, channelId byte) model.Provider[Model] {
		return func() (Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetChannelRegistry().ChannelServer(t.Id().String(), worldId, channelId)
		}
	}
}

func GetById(_ logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte) (Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte) (Model, error) {
		return func(worldId byte, channelId byte) (Model, error) {
			return ByIdProvider(ctx)(worldId, channelId)()
		}
	}
}

func Register(_ logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
		return func(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetChannelRegistry().Register(t.Id().String(), worldId, channelId, ipAddress, port), nil
		}
	}
}

func Unregister(_ logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte) error {
	return func(ctx context.Context) func(worldId byte, channelId byte) error {
		return func(worldId byte, channelId byte) error {
			t := tenant.MustFromContext(ctx)
			GetChannelRegistry().RemoveByWorldAndChannel(t.Id().String(), worldId, channelId)
			return nil
		}
	}
}

func RequestStatus(l logrus.FieldLogger) func(ctx context.Context) func(c configuration.Model) {
	return func(ctx context.Context) func(c configuration.Model) {
		return func(c configuration.Model) {
			for _, sc := range c.Data.Attributes.Servers {
				t, _ := tenant.Create(uuid.MustParse(sc.Tenant), "", 0, 0)
				_ = producer.ProviderImpl(l)(tenant.WithContext(ctx, t))(EnvCommandTopicChannelStatus)(emitChannelServerStatusCommand(t))
			}
		}
	}
}
