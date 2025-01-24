package list

import (
	consumer2 "atlas-buddies/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("buddy_list_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateBuddyListCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestBuddyAddCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestBuddyDeleteCommand(db))))
		}
	}
}

func handleCreateBuddyListCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[createCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[createCommandBody]) {
		if c.Type != CommandTypeCreate {
			return
		}
		_, err := list.Create(l)(ctx)(db)(c.CharacterId, c.Body.Capacity)
		if err != nil {
			l.WithError(err).Errorf("Error creating buddy list for character [%d].", c.CharacterId)
		}
	}
}

func handleRequestBuddyAddCommand(db *gorm.DB) message.Handler[command[requestAddBuddyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestAddBuddyCommandBody]) {
		if c.Type != CommandTypeRequestAdd {
			return
		}
		err := list.RequestAdd(l)(ctx)(db)(c.CharacterId, c.WorldId, c.Body.CharacterId, c.Body.Group)
		if err != nil {
			l.WithError(err).Errorf("Error attempting to add [%d] to character [%d] buddy list.", c.Body.CharacterId, c.CharacterId)
		}
	}
}

func handleRequestBuddyDeleteCommand(db *gorm.DB) message.Handler[command[requestDeleteBuddyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestDeleteBuddyCommandBody]) {
		if c.Type != CommandTypeRequestDelete {
			return
		}
		err := list.RequestDelete(l)(ctx)(db)(c.CharacterId, c.WorldId, c.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Error attempting to delete [%d] to character [%d] buddy list.", c.Body.CharacterId, c.CharacterId)
		}
	}
}
