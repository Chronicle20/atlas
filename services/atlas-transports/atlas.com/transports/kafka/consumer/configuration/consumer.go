package configuration

import (
	"atlas-transports/instance"
	instanceConfig "atlas-transports/instance/config"
	consumer2 "atlas-transports/kafka/consumer"
	configuration2 "atlas-transports/kafka/message/configuration"
	"atlas-transports/transport"
	"atlas-transports/transport/config"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("configuration_status_event")(configuration2.EnvEventTopicConfigurationStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(configuration2.EnvEventTopicConfigurationStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleConfigurationStatus))); err != nil {
			return err
		}
		return nil
	}
}

func handleConfigurationStatus(l logrus.FieldLogger, ctx context.Context, e configuration2.StatusEvent) {
	t := tenant.MustFromContext(ctx)
	if t.Id() == uuid.Nil {
		// consumer.TenantHeaderParser installs the ZERO tenant when the
		// message carries no tenant headers, so without this guard a
		// header-less event would ClearTenant() and reload nothing —
		// silently emptying a healthy registry. Refusing to act makes
		// the nil-tenant signature a loud regression instead of a quiet
		// data-loss event.
		l.Errorf("Configuration-status event [%s] for resource [%s] arrived without tenant headers; skipping reload.", e.Type, e.ResourceId)
		return
	}

	switch e.ResourceType {
	case "route", "vessel":
		l.Infof("Configuration [%s] event [%s] for resource [%s], reloading scheduled routes for tenant [%s].", e.ResourceType, e.Type, e.ResourceId, t.Id())

		// Load BEFORE clearing: the reload is a full replace, so a load
		// failure after a clear would leave the tenant with no routes.
		routes, sharedVessels, err := config.NewProcessor(l, ctx).LoadConfigurationsForTenant(t)
		if err != nil {
			l.WithError(err).Errorf("Failed to reload configurations for tenant [%s]; leaving the scheduled route registry untouched.", t.Id())
			return
		}
		tp := transport.NewProcessor(l, ctx)
		tp.ClearTenant()
		_ = tp.AddTenant(routes, sharedVessels)
	case "instance-route":
		l.Infof("Configuration [%s] event [%s] for resource [%s], reloading instance routes for tenant [%s].", e.ResourceType, e.Type, e.ResourceId, t.Id())

		instanceRoutes, err := instanceConfig.NewProcessor(l, ctx).LoadConfigurationsForTenant(t)
		if err != nil {
			l.WithError(err).Errorf("Failed to reload instance route configurations for tenant [%s]; leaving the instance route registry untouched.", t.Id())
			return
		}
		ip := instance.NewProcessor(l, ctx)
		ip.ClearTenant()
		ip.AddTenant(instanceRoutes)
	default:
		l.Warnf("Unhandled configuration resource type [%s].", e.ResourceType)
	}
}
