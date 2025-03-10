package pet

import (
	consumer2 "atlas-pets/kafka/consumer"
	"atlas-pets/pet"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("pet_movement_command")(EnvCommandTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopicMovement)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementCommand(db))))
		}
	}
}

func handleMovementCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c movementCommand) {
	return func(l logrus.FieldLogger, ctx context.Context, c movementCommand) {
		m := _map.NewModel(world.Id(c.WorldId))(channel.Id(c.ChannelId))(_map.Id(c.MapId))
		err := pet.Move(l)(ctx)(db)(c.PetId)(m)(c.CharacterId)(c.Movement)
		if err != nil {
			l.WithError(err).Errorf("Error processing movement for pet [%d].", c.PetId)
		}
	}
}
