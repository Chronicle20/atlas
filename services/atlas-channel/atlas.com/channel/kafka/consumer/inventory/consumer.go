package inventory

import (
	"atlas-channel/asset"
	consumer2 "atlas-channel/kafka/consumer"
	inventory2 "atlas-channel/kafka/message/inventory"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_inventory_changed_event")(inventory2.EnvEventInventoryChanged)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(inventory2.EnvEventInventoryChanged)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryAttributeUpdateEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryItemReservationCancelledEvent(sc, wp))))
			}
		}
	}
}

func handleInventoryAttributeUpdateEvent(sc server.Model, wp writer.Producer) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAttributeUpdateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAttributeUpdateBody]) {
		if e.Type != inventory2.ChangedTypeUpdateAttribute {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		eqp := asset.NewBuilder[asset.EquipableReferenceData](0, e.Body.ItemId, 0, asset.ReferenceTypeEquipable).
			SetSlot(e.Slot).
			SetExpiration(e.Body.Expiration).
			SetReferenceData(asset.NewEquipableReferenceDataBuilder().
				SetStrength(e.Body.Strength).
				SetDexterity(e.Body.Dexterity).
				SetIntelligence(e.Body.Intelligence).
				SetLuck(e.Body.Luck).
				SetHp(e.Body.HP).
				SetMp(e.Body.MP).
				SetWeaponAttack(e.Body.WeaponAttack).
				SetMagicAttack(e.Body.MagicAttack).
				SetWeaponDefense(e.Body.WeaponDefense).
				SetMagicDefense(e.Body.MagicDefense).
				SetAccuracy(e.Body.Accuracy).
				SetAvoidability(e.Body.Avoidability).
				SetHands(e.Body.Hands).
				SetSpeed(e.Body.Speed).
				SetJump(e.Body.Jump).
				SetSlots(e.Body.Slots).
				SetOwnerId(0).
				SetLocked(e.Body.Locked).
				SetSpikes(e.Body.Spikes).
				SetKarmaUsed(e.Body.KarmaUsed).
				SetCold(e.Body.Cold).
				SetCanBeTraded(e.Body.CanBeTraded).
				SetLevelType(e.Body.LevelType).
				SetLevel(e.Body.Level).
				SetExperience(e.Body.Experience).
				SetHammersApplied(e.Body.HammersApplied).
				Build()).
			Build()

		so := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryRefreshEquipable(sc.Tenant())(eqp))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, so)
		if err != nil {
			l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", e.Body.ItemId, e.Slot, e.CharacterId)
		}
	}
}

func handleInventoryItemReservationCancelledEvent(sc server.Model, wp writer.Producer) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemRemoveBody]) {
		if e.Type != inventory2.ChangedTypeReservationCancelled {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true)))
	}
}
