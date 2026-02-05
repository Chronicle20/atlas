package session

import (
	"atlas-effective-stats/character"
	consumer2 "atlas-effective-stats/kafka/consumer"
	message "atlas-effective-stats/kafka/message/session"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("session_status")(message.EnvEventTopicSessionStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(message.EnvEventTopicSessionStatus)()
		_, _ = rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleSessionStatus(l))))
	}
}

func handleSessionStatus(l logrus.FieldLogger) kafkaMessage.Handler[message.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e message.StatusEvent) {
		if e.Type == message.EventSessionStatusTypeCreated {
			handleSessionCreated(l, ctx, e)
		} else if e.Type == message.EventSessionStatusTypeDestroyed {
			handleSessionDestroyed(l, ctx, e)
		}
	}
}

func handleSessionCreated(l logrus.FieldLogger, ctx context.Context, e message.StatusEvent) {
	// Only process channel events (character is actually logged in and on a map)
	if e.Issuer != message.EventSessionStatusIssuerChannel {
		return
	}

	l.Debugf("Session created for character [%d] on world [%d] channel [%d].", e.CharacterId, e.WorldId, e.ChannelId)

	// Initialize effective stats for this character
	ch := channel.NewModel(e.WorldId, e.ChannelId)
	if err := character.InitializeCharacter(l, ctx, e.CharacterId, ch); err != nil {
		l.WithError(err).Warnf("Failed to initialize effective stats for character [%d].", e.CharacterId)
	}
}

func handleSessionDestroyed(l logrus.FieldLogger, ctx context.Context, e message.StatusEvent) {
	l.Debugf("Session destroyed for character [%d].", e.CharacterId)

	// Remove character from the effective stats registry
	p := character.NewProcessor(l, ctx)
	p.RemoveCharacter(e.CharacterId)
}
