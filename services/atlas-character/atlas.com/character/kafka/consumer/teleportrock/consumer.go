package teleportrock

import (
	consumer2 "atlas-character/kafka/consumer"
	teleportrock2 "atlas-character/kafka/message/teleportrock"
	"atlas-character/teleport_rock"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("teleport_rock_command")(teleportrock2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(teleportrock2.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAddMap(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoveMap(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleAddMap(db *gorm.DB) message.Handler[teleportrock2.Command[teleportrock2.AddMapCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c teleportrock2.Command[teleportrock2.AddMapCommandBody]) {
		if c.Type != teleportrock2.CommandAddMap {
			return
		}
		_ = teleport_rock.NewProcessor(l, ctx, db).AddMapAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.MapId, c.Body.Vip)
	}
}

func handleRemoveMap(db *gorm.DB) message.Handler[teleportrock2.Command[teleportrock2.RemoveMapCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c teleportrock2.Command[teleportrock2.RemoveMapCommandBody]) {
		if c.Type != teleportrock2.CommandRemoveMap {
			return
		}
		_ = teleport_rock.NewProcessor(l, ctx, db).RemoveMapAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.MapId, c.Body.Vip)
	}
}
