package equipable

import (
	"atlas-character/equipable"
	"atlas-character/inventory"
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

		updates := []equipable.Updater{
			equipable.AddStrength(c.Body.Strength),
			equipable.AddDexterity(c.Body.Dexterity),
			equipable.AddIntelligence(c.Body.Intelligence),
			equipable.AddLuck(c.Body.Luck),
			equipable.AddHP(c.Body.HP),
			equipable.AddMP(c.Body.MP),
			equipable.AddWeaponAttack(c.Body.WeaponAttack),
			equipable.AddMagicAttack(c.Body.MagicAttack),
			equipable.AddWeaponDefense(c.Body.WeaponDefense),
			equipable.AddMagicDefense(c.Body.MagicDefense),
			equipable.AddAccuracy(c.Body.Accuracy),
			equipable.AddAvoidability(c.Body.Avoidability),
			equipable.AddHands(c.Body.Hands),
			equipable.AddSpeed(c.Body.Speed),
			equipable.AddJump(c.Body.Jump),
			equipable.AddSlots(c.Body.Slots),
		}
		_ = inventory.UpdateEquip(l)(ctx)(db)(c.CharacterId, c.Slot, updates...)
	}
}
