package configuration

import (
	"atlas-transports/instance"
	instanceConfig "atlas-transports/instance/config"
	consumer2 "atlas-transports/kafka/consumer"
	configuration2 "atlas-transports/kafka/message/configuration"
	"atlas-transports/transport"
	"atlas-transports/transport/config"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("configuration_status_event")(configuration2.EnvEventTopicConfigurationStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(configuration2.EnvEventTopicConfigurationStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleConfigurationStatus)))
	}
}

func handleConfigurationStatus(l logrus.FieldLogger, ctx context.Context, e configuration2.StatusEvent) {
	switch e.ResourceType {
	case "route", "vessel":
		l.Infof("Configuration [%s] event [%s] for resource [%s], reloading scheduled routes for tenant [%s].", e.ResourceType, e.Type, e.ResourceId, e.TenantId)
		t := tenant.MustFromContext(ctx)

		tp := transport.NewProcessor(l, ctx)
		tp.ClearTenant()

		routes, sharedVessels, err := config.NewProcessor(l, ctx).LoadConfigurationsForTenant(t)
		if err != nil {
			l.WithError(err).Errorf("Failed to reload configurations for tenant [%s].", e.TenantId)
			return
		}
		_ = tp.AddTenant(routes, sharedVessels)
	case "instance-route":
		l.Infof("Configuration [%s] event [%s] for resource [%s], reloading instance routes for tenant [%s].", e.ResourceType, e.Type, e.ResourceId, e.TenantId)
		t := tenant.MustFromContext(ctx)

		ip := instance.NewProcessor(l, ctx)
		ip.ClearTenant()

		instanceRoutes, err := instanceConfig.NewProcessor(l, ctx).LoadConfigurationsForTenant(t)
		if err != nil {
			l.WithError(err).Errorf("Failed to reload instance route configurations for tenant [%s].", e.TenantId)
			return
		}
		ip.AddTenant(instanceRoutes)
	default:
		l.Warnf("Unhandled configuration resource type [%s].", e.ResourceType)
	}
}
