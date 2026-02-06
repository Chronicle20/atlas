package asset

import (
	"atlas-equipables/equipable"
	consumer2 "atlas-equipables/kafka/consumer"
	"atlas-equipables/kafka/message/asset"
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
			rf(consumer2.NewConfig(l)("asset_status_event")(asset.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(asset.EnvEventTopicStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovedEvent(db))))
		}
	}
}

func handleMovedEvent(db *gorm.DB) message.Handler[asset.StatusEvent[asset.MovedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.MovedStatusEventBody]) {
		// Only process MOVED events
		if e.Type != asset.StatusEventTypeMoved {
			return
		}

		oldSlot := e.Body.OldSlot
		newSlot := e.Slot
		referenceId := e.AssetId // This is the equipable ID

		p := equipable.NewProcessor(l, ctx, db)

		if asset.IsEquipEvent(oldSlot, newSlot) {
			l.Debugf("Equipment [%d] equipped (slot %d -> %d). Setting equippedSince.", referenceId, oldSlot, newSlot)
			err := p.MarkEquipped(referenceId)
			if err != nil {
				l.WithError(err).Warnf("Failed to mark equipment [%d] as equipped.", referenceId)
			}
			return
		}

		if asset.IsUnequipEvent(oldSlot, newSlot) {
			l.Debugf("Equipment [%d] unequipped (slot %d -> %d). Clearing equippedSince.", referenceId, oldSlot, newSlot)
			err := p.MarkUnequipped(referenceId)
			if err != nil {
				l.WithError(err).Warnf("Failed to mark equipment [%d] as unequipped.", referenceId)
			}
			return
		}

		// Other moves (inventory reorganization) don't affect equipped status
	}
}
