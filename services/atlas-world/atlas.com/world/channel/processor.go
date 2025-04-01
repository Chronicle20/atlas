package channel

import (
	tenant2 "atlas-world/configuration/tenant"
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

func RequestStatus(l logrus.FieldLogger) func(ctx context.Context) func(tenantId uuid.UUID) model.Operator[tenant2.RestModel] {
	return func(ctx context.Context) func(tenantId uuid.UUID) model.Operator[tenant2.RestModel] {
		return func(tenantId uuid.UUID) model.Operator[tenant2.RestModel] {
			return func(config tenant2.RestModel) error {
				t, err := tenant.Create(uuid.MustParse(config.Id), config.Region, config.MajorVersion, config.MinorVersion)
				if err != nil {
					return err
				}
				l.Debugf("Requesting status of channels for tenant [%s].", t.String())
				err = producer.ProviderImpl(l)(tenant.WithContext(ctx, t))(EnvCommandTopicChannelStatus)(emitChannelServerStatusCommand(t))
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
}
