package equipable

import (
	"atlas-equipables/equipable"
	consumer2 "atlas-equipables/kafka/consumer"
	equipable2 "atlas-equipables/kafka/message/equipable"
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
			rf(consumer2.NewConfig(l)("equipable_command")(equipable2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(equipable2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeCommand(db))))
		}
	}
}

func handleChangeCommand(db *gorm.DB) message.Handler[equipable2.Command[equipable2.AttributeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c equipable2.Command[equipable2.AttributeBody]) {
		if c.Type != equipable2.CommandChange {
			return
		}
		i := equipable.NewBuilder(c.Id).
			SetStrength(c.Body.Strength).
			SetDexterity(c.Body.Dexterity).
			SetIntelligence(c.Body.Intelligence).
			SetLuck(c.Body.Luck).
			SetHp(c.Body.HP).
			SetMp(c.Body.MP).
			SetWeaponAttack(c.Body.WeaponAttack).
			SetMagicAttack(c.Body.MagicAttack).
			SetWeaponDefense(c.Body.WeaponDefense).
			SetMagicDefense(c.Body.MagicDefense).
			SetAccuracy(c.Body.Accuracy).
			SetAvoidability(c.Body.Avoidability).
			SetHands(c.Body.Hands).
			SetSpeed(c.Body.Speed).
			SetJump(c.Body.Jump).
			SetSlots(c.Body.Slots).
			SetOwnerName(c.Body.OwnerName).
			SetLocked(c.Body.Locked).
			SetSpikes(c.Body.Spikes).
			SetKarmaUsed(c.Body.KarmaUsed).
			SetCold(c.Body.Cold).
			SetCanBeTraded(c.Body.CanBeTraded).
			SetLevelType(c.Body.LevelType).
			SetLevel(c.Body.Level).
			SetExperience(c.Body.Experience).
			SetHammersApplied(c.Body.HammersApplied).
			SetExpiration(c.Body.Expiration).
			Build()
		_, _ = equipable.NewProcessor(l, ctx, db).UpdateAndEmit(i)
	}
}
