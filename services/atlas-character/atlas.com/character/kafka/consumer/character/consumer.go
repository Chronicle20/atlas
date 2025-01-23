package character

import (
	"atlas-character/character"
	consumer2 "atlas-character/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("character_movement_command")(EnvCommandTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeMap(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestChangeMeso(db))))
			t, _ = topic.EnvProvider(l)(EnvCommandTopicMovement)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementEvent)))
		}
	}
}

func handleChangeMap(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c commandEvent[changeMapBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c commandEvent[changeMapBody]) {
		if c.Type != CommandChangeMap {
			return
		}

		err := character.ChangeMap(l, db, ctx)(c.CharacterId, c.WorldId, c.Body.ChannelId, c.Body.MapId, c.Body.PortalId)
		if err != nil {
			l.WithError(err).Errorf("Unable to change character [%d] map.", c.CharacterId)
		}
	}
}

func handleRequestChangeMeso(db *gorm.DB) message.Handler[commandEvent[requestChangeMesoBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c commandEvent[requestChangeMesoBody]) {
		if c.Type != CommandRequestChangeMeso {
			return
		}

		_ = character.RequestChangeMeso(l)(ctx)(db)(c.CharacterId, c.Body.Amount)
	}
}

func handleMovementEvent(l logrus.FieldLogger, ctx context.Context, c movementCommand) {
	err := character.Move(l)(ctx)(c.CharacterId)(c.WorldId)(c.ChannelId)(c.MapId)(c.Movement)
	if err != nil {
		l.WithError(err).Errorf("Error processing movement for character [%d].", c.CharacterId)
	}
}
