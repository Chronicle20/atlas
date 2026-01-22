package asset

import (
	consumer2 "atlas-quest/kafka/consumer"
	"atlas-quest/kafka/message/asset"
	"atlas-quest/quest"
	"context"
	"strconv"

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
			rf(consumer2.NewConfig(l)("asset_status")(asset.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(asset.EnvEventTopicStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetCreatedEvent(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeletedEvent(db))))
		}
	}
}

func handleAssetCreatedEvent(db *gorm.DB) message.Handler[asset.StatusEvent[asset.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.CreatedStatusEventBody]) {
		if e.Type != asset.StatusEventTypeCreated {
			return
		}

		if e.CharacterId == 0 {
			return
		}

		// Get all started quests for this character
		quests, err := quest.NewProcessor(l, ctx, db).GetByCharacterIdAndState(e.CharacterId, quest.StateStarted)
		if err != nil {
			l.WithError(err).Debugf("Unable to get started quests for character [%d].", e.CharacterId)
			return
		}

		// For each quest, check if it tracks this item and update progress
		// The infoNumber for item collection is typically the itemId (TemplateId)
		for _, q := range quests {
			// Check if this quest tracks this item (by having progress with infoNumber = itemId)
			if p, found := q.GetProgress(e.TemplateId); found {
				// Increment the item count
				currentCount := parseProgress(p.Progress())
				quantity := e.Body.Quantity
				if quantity == 0 {
					quantity = 1
				}
				newCount := currentCount + quantity
				err = quest.NewProcessor(l, ctx, db).SetProgress(e.CharacterId, q.QuestId(), e.TemplateId, strconv.Itoa(int(newCount)))
				if err != nil {
					l.WithError(err).Errorf("Unable to update item progress for quest [%d] character [%d].", q.QuestId(), e.CharacterId)
				} else {
					l.Debugf("Updated item [%d] progress for quest [%d] character [%d]: %d -> %d.", e.TemplateId, q.QuestId(), e.CharacterId, currentCount, newCount)
				}
			}
		}
	}
}

func handleAssetDeletedEvent(db *gorm.DB) message.Handler[asset.StatusEvent[asset.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.DeletedStatusEventBody]) {
		if e.Type != asset.StatusEventTypeDeleted {
			return
		}

		if e.CharacterId == 0 {
			return
		}

		// Get all started quests for this character
		quests, err := quest.NewProcessor(l, ctx, db).GetByCharacterIdAndState(e.CharacterId, quest.StateStarted)
		if err != nil {
			l.WithError(err).Debugf("Unable to get started quests for character [%d].", e.CharacterId)
			return
		}

		// For each quest, check if it tracks this item and update progress
		for _, q := range quests {
			if p, found := q.GetProgress(e.TemplateId); found {
				// Decrement the item count (but not below 0)
				currentCount := parseProgress(p.Progress())
				var newCount uint32
				if currentCount > 0 {
					newCount = currentCount - 1
				}
				err = quest.NewProcessor(l, ctx, db).SetProgress(e.CharacterId, q.QuestId(), e.TemplateId, strconv.Itoa(int(newCount)))
				if err != nil {
					l.WithError(err).Errorf("Unable to update item progress for quest [%d] character [%d].", q.QuestId(), e.CharacterId)
				} else {
					l.Debugf("Updated item [%d] progress (deleted) for quest [%d] character [%d]: %d -> %d.", e.TemplateId, q.QuestId(), e.CharacterId, currentCount, newCount)
				}
			}
		}
	}
}

func parseProgress(progress string) uint32 {
	if progress == "" {
		return 0
	}
	val, err := strconv.Atoi(progress)
	if err != nil {
		return 0
	}
	return uint32(val)
}
