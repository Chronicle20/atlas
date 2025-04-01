package channel

import (
	"atlas-channel/configuration"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
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

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		ctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(context.Background(), "teardown")
		defer span.End()

		sc, err := configuration.GetServiceConfig()
		if err != nil {
			return
		}
		l.Debugf("Unregistering service world channels.")
		_ = model.ForEachSlice(model.FixedProvider(sc.Tenants), teardown(l)(ctx))
	}
}

func teardown(l logrus.FieldLogger) func(ctx context.Context) model.Operator[configuration.ChannelTenantRestModel] {
	return func(ctx context.Context) model.Operator[configuration.ChannelTenantRestModel] {
		return func(config configuration.ChannelTenantRestModel) error {
			tenantId := uuid.MustParse(config.Id)
			tc, err := configuration.GetTenantConfig(tenantId)
			if err != nil {
				return err
			}
			t, err := tenant.Create(uuid.MustParse(tc.Id), tc.Region, tc.MajorVersion, tc.MinorVersion)
			if err != nil {
				return err
			}
			tctx := tenant.WithContext(ctx, t)
			for _, w := range config.Worlds {
				for _, c := range w.Channels {
					err = Unregister(l)(tctx)(world.Id(w.Id), channel.Id(c.Id))
					if err != nil {
						l.WithError(err).Errorf("Unable to unregister world [%d] channel [%d] for tenant [%s].", w.Id, c.Id, t.String())
					}
				}
			}
			return nil
		}
	}
}
