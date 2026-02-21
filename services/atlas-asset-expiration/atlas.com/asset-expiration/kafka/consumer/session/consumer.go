package session

import (
	"atlas-asset-expiration/character"
	consumer2 "atlas-asset-expiration/kafka/consumer"
	message "atlas-asset-expiration/kafka/message/session"
	"atlas-asset-expiration/kafka/producer"
	"atlas-asset-expiration/session"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("session_status")(message.EnvEventTopicSessionStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(message.EnvEventTopicSessionStatus)()
		if _, err := rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleSessionStatus(l)))); err != nil {
			return err
		}
		return nil
	}
}

func handleSessionStatus(_ logrus.FieldLogger) kafkaMessage.Handler[message.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e message.StatusEvent) {
		if e.Type == message.EventSessionStatusTypeCreated {
			handleSessionCreated(l, ctx, e)
		} else if e.Type == message.EventSessionStatusTypeDestroyed {
			handleSessionDestroyed(l, ctx, e)
		}
	}
}

func handleSessionCreated(l logrus.FieldLogger, ctx context.Context, e message.StatusEvent) {
	// Only process login events
	if e.Issuer != message.EventSessionStatusIssuerLogin {
		return
	}

	l.Infof("Session created for character [%d], account [%d], world [%d].", e.CharacterId, e.AccountId, e.WorldId)

	// Extract tenant info from context
	t, err := tenant.FromContext(ctx)()
	if err != nil {
		l.WithError(err).Warnf("Failed to extract tenant from context for character [%d].", e.CharacterId)
		return
	}

	// Add to session tracker for periodic checks
	ch := channel.NewModel(e.WorldId, e.ChannelId)
	session.GetTracker().Add(e.CharacterId, e.AccountId, ch, t.Id(), t.Region(), t.MajorVersion(), t.MinorVersion())

	// Immediate expiration check on login
	pp := producer.ProviderImpl(l)(ctx)
	character.CheckAndExpire(l)(pp)(ctx)(e.CharacterId, e.AccountId, e.WorldId)
}

func handleSessionDestroyed(l logrus.FieldLogger, _ context.Context, e message.StatusEvent) {
	l.Debugf("Session destroyed for character [%d].", e.CharacterId)

	// Remove from session tracker
	session.GetTracker().Remove(e.CharacterId)
}
