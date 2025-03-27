package equipable

import (
	"atlas-character/equipable"
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
			rf(consumer2.NewConfig(l)("equipable_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeCommand(db))))
		}
	}
}

func handleChangeCommand(db *gorm.DB) message.Handler[command[changeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[changeBody]) {
		if c.Type != CommandChange {
			return
		}
		_ = equipable.Update(l)(db)(ctx)(c.CharacterId, c.ItemId, c.Slot, c.Body.Strength, c.Body.Dexterity, c.Body.Intelligence, c.Body.Luck, c.Body.HP, c.Body.MP, c.Body.WeaponAttack, c.Body.MagicAttack, c.Body.WeaponDefense, c.Body.MagicDefense, c.Body.Accuracy, c.Body.Avoidability, c.Body.Hands, c.Body.Speed, c.Body.Jump, c.Body.Speed)
	}
}
