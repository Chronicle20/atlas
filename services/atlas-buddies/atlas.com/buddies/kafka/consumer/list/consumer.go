package list

import (
	consumer2 "atlas-buddies/kafka/consumer"
	list2 "atlas-buddies/kafka/message/list"
	"atlas-buddies/list"
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
			rf(consumer2.NewConfig(l)("buddy_list_command")(list2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(list2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateBuddyListCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestBuddyAddCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestBuddyDeleteCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleIncreaseCapacityCommand(db))))
		}
	}
}

func handleCreateBuddyListCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c list2.Command[list2.CreateCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c list2.Command[list2.CreateCommandBody]) {
		if c.Type != list2.CommandTypeCreate {
			return
		}
		_, err := list.NewProcessor(l, ctx, db).Create(c.CharacterId, c.Body.Capacity)
		if err != nil {
			l.WithError(err).Errorf("Error creating buddy list for character [%d].", c.CharacterId)
		}
	}
}

func handleRequestBuddyAddCommand(db *gorm.DB) message.Handler[list2.Command[list2.RequestAddBuddyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c list2.Command[list2.RequestAddBuddyCommandBody]) {
		if c.Type != list2.CommandTypeRequestAdd {
			return
		}
		err := list.NewProcessor(l, ctx, db).RequestAddBuddyAndEmit(c.CharacterId, c.WorldId, c.Body.CharacterId, c.Body.Group)
		if err != nil {
			l.WithError(err).Errorf("Error attempting to add [%d] to character [%d] buddy list.", c.Body.CharacterId, c.CharacterId)
		}
	}
}

func handleRequestBuddyDeleteCommand(db *gorm.DB) message.Handler[list2.Command[list2.RequestDeleteBuddyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c list2.Command[list2.RequestDeleteBuddyCommandBody]) {
		if c.Type != list2.CommandTypeRequestDelete {
			return
		}
		err := list.NewProcessor(l, ctx, db).RequestDeleteBuddyAndEmit(c.CharacterId, c.WorldId, c.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Error attempting to delete [%d] to character [%d] buddy list.", c.Body.CharacterId, c.CharacterId)
		}
	}
}

func handleIncreaseCapacityCommand(db *gorm.DB) message.Handler[list2.Command[list2.IncreaseCapacityCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c list2.Command[list2.IncreaseCapacityCommandBody]) {
		if c.Type != list2.CommandTypeIncreaseCapacity {
			return
		}
		err := list.NewProcessor(l, ctx, db).IncreaseCapacityAndEmit(c.CharacterId, c.WorldId, c.Body.NewCapacity)
		if err != nil {
			l.WithError(err).Errorf("Failed to increase buddy list capacity for character [%d].", c.CharacterId)
		}
	}
}
