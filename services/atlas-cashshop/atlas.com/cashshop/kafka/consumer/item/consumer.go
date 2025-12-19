package item

import (
	itemModel "atlas-cashshop/cashshop/item"
	consumer2 "atlas-cashshop/kafka/consumer"
	itemMessage "atlas-cashshop/kafka/message/item"
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
			rf(consumer2.NewConfig(l)("item_command")(itemMessage.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(itemMessage.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandCreate(db))))
		}
	}
}

func handleCommandCreate(db *gorm.DB) message.Handler[itemMessage.Command[itemMessage.CreateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, command itemMessage.Command[itemMessage.CreateCommandBody]) {
		if command.Type != itemMessage.CommandCreate {
			return
		}

		l.Debugf("Handling create item command for character %d", command.CharacterId)

		_, err := itemModel.NewProcessor(l, ctx, db).CreateAndEmit(
			command.Body.TemplateId,
			command.Body.Quantity,
			command.Body.PurchasedBy,
		)

		if err != nil {
			l.WithError(err).Errorf("Error creating item.")
			return
		}

		l.Debugf("Successfully created item for character %d", command.CharacterId)
	}
}
