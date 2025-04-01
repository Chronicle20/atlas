package channel

import (
	tenant2 "atlas-world/configuration/tenant"
	channel2 "atlas-world/kafka/message/channel"
	"atlas-world/kafka/producer"
	channel3 "atlas-world/kafka/producer/channel"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

func AllProvider(ctx context.Context) model.Provider[[]Model] {
	return model.FixedProvider(GetChannelRegistry().ChannelServers(tenant.MustFromContext(ctx)))
}

func ByWorldProvider(ctx context.Context) func(worldId byte) model.Provider[[]Model] {
	return func(worldId byte) model.Provider[[]Model] {
		return model.FilteredProvider[Model](AllProvider(ctx), model.Filters(ByWorldFilter(worldId)))
	}
}

func ByWorldFilter(id byte) model.Filter[Model] {
	return func(m Model) bool {
		return m.WorldId() == id
	}
}

func GetByWorld(ctx context.Context) func(worldId byte) ([]Model, error) {
	return func(worldId byte) ([]Model, error) {
		return ByWorldProvider(ctx)(worldId)()
	}
}

func ByIdProvider(ctx context.Context) func(worldId byte, channelId byte) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(worldId byte, channelId byte) model.Provider[Model] {
		cs, err := GetChannelRegistry().ChannelServer(t, worldId, channelId)
		if err != nil {
			return model.ErrorProvider[Model](err)
		}
		return model.FixedProvider(cs)
	}
}

func GetById(ctx context.Context) func(worldId byte, channelId byte) (Model, error) {
	return func(worldId byte, channelId byte) (Model, error) {
		return ByIdProvider(ctx)(worldId, channelId)()
	}
}

func Register(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, channelId byte, ipAddress string, port int) (Model, error) {
			l.Debugf("Registering world [%d] channel [%d] for tenant [%s].", worldId, channelId, t.String())
			m := Model{
				id:        uuid.New(),
				worldId:   worldId,
				channelId: channelId,
				ipAddress: ipAddress,
				port:      port,
				createdAt: time.Now(),
			}
			return GetChannelRegistry().Register(t, m), nil
		}
	}
}

func Unregister(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte) error {
	return func(ctx context.Context) func(worldId byte, channelId byte) error {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, channelId byte) error {
			l.Debugf("Unregistering world [%d] channel [%d] for tenant [%s].", worldId, channelId, t.String())
			return GetChannelRegistry().RemoveByWorldAndChannel(t, worldId, channelId)
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
				err = producer.ProviderImpl(l)(tenant.WithContext(ctx, t))(channel2.EnvCommandTopic)(channel3.StatusCommandProvider(t))
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
}
