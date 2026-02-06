package quest

import (
	dataQuest "atlas-quest/data/quest"
	consumer2 "atlas-quest/kafka/consumer"
	quest2 "atlas-quest/kafka/message/quest"
	"atlas-quest/kafka/message/saga"
	sagaProducer "atlas-quest/kafka/producer/saga"
	"atlas-quest/quest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
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
			rf(consumer2.NewConfig(l)("quest_command")(quest2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(quest2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStartQuestCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompleteQuestCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleForfeitQuestCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdateProgressCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRestoreItemCommand())))
		}
	}
}

func handleStartQuestCommand(db *gorm.DB) message.Handler[quest2.Command[quest2.StartCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c quest2.Command[quest2.StartCommandBody]) {
		if c.Type != quest2.CommandTypeStart {
			return
		}
		// Use Force flag to determine whether to skip validation
		// When Force=true, skip requirement checks
		// When Force=false (default), validate start requirements before starting
		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).Build()
		_, _, err := quest.NewProcessor(l, ctx, db).Start(c.TransactionId, c.CharacterId, c.Body.QuestId, f, c.Body.Force)
		if err != nil {
			l.WithError(err).Errorf("Error starting quest [%d] for character [%d].", c.Body.QuestId, c.CharacterId)
		}
	}
}

func handleCompleteQuestCommand(db *gorm.DB) message.Handler[quest2.Command[quest2.CompleteCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c quest2.Command[quest2.CompleteCommandBody]) {
		if c.Type != quest2.CommandTypeComplete {
			return
		}
		processor := quest.NewProcessor(l, ctx, db)
		// Use Force flag to determine whether to skip validation
		// When Force=true, skip requirement checks (forceCompleteQuest behavior)
		// When Force=false, validate end requirements before completing
		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).Build()
		nextQuestId, err := processor.Complete(c.TransactionId, c.CharacterId, c.Body.QuestId, f, c.Body.Force)
		if err != nil {
			l.WithError(err).Errorf("Error completing quest [%d] for character [%d].", c.Body.QuestId, c.CharacterId)
			return
		}

		// Handle quest chain - auto-start next quest if present
		if nextQuestId > 0 {
			l.Infof("Quest chain detected: starting next quest [%d] for character [%d].", nextQuestId, c.CharacterId)
			// Use the same transactionId for chained quests so the saga can track them
			_, err = processor.StartChained(c.TransactionId, c.CharacterId, nextQuestId, f)
			if err != nil {
				l.WithError(err).Errorf("Error starting chained quest [%d] for character [%d].", nextQuestId, c.CharacterId)
			}
		}
	}
}

func handleForfeitQuestCommand(db *gorm.DB) message.Handler[quest2.Command[quest2.ForfeitCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c quest2.Command[quest2.ForfeitCommandBody]) {
		if c.Type != quest2.CommandTypeForfeit {
			return
		}
		err := quest.NewProcessor(l, ctx, db).Forfeit(c.TransactionId, c.CharacterId, c.Body.QuestId)
		if err != nil {
			l.WithError(err).Errorf("Error forfeiting quest [%d] for character [%d].", c.Body.QuestId, c.CharacterId)
		}
	}
}

func handleUpdateProgressCommand(db *gorm.DB) message.Handler[quest2.Command[quest2.UpdateProgressCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c quest2.Command[quest2.UpdateProgressCommandBody]) {
		if c.Type != quest2.CommandTypeUpdateProgress {
			return
		}
		err := quest.NewProcessor(l, ctx, db).SetProgress(c.TransactionId, c.CharacterId, c.Body.QuestId, c.Body.InfoNumber, c.Body.Progress)
		if err != nil {
			l.WithError(err).Errorf("Error updating progress for quest [%d] for character [%d].", c.Body.QuestId, c.CharacterId)
		}
	}
}

func handleRestoreItemCommand() message.Handler[quest2.Command[quest2.RestoreItemCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c quest2.Command[quest2.RestoreItemCommandBody]) {
		if c.Type != quest2.CommandTypeRestoreItem {
			return
		}

		// Look up the quest definition
		questDef, err := dataQuest.NewProcessor(l, ctx).GetQuestDefinition(c.Body.QuestId)
		if err != nil {
			l.WithError(err).Errorf("Error getting quest definition [%d] for item restoration.", c.Body.QuestId)
			return
		}

		// Check if the quest's StartActions.Items contains the requested itemId
		var foundItem *dataQuest.ItemReward
		for _, item := range questDef.StartActions.Items {
			if item.Id == c.Body.ItemId && item.Count > 0 {
				foundItem = &item
				break
			}
		}

		if foundItem == nil {
			l.Warnf("Item [%d] not found in quest [%d] start actions for character [%d].", c.Body.ItemId, c.Body.QuestId, c.CharacterId)
			return
		}

		// Build and emit a saga to award the item
		builder := sagaProducer.NewBuilder(saga.QuestRestoreItem, fmt.Sprintf("quest_%d_restore_item_%d", c.Body.QuestId, c.Body.ItemId))
		builder.AddAwardItem(c.CharacterId, foundItem.Id, uint32(foundItem.Count))

		if builder.HasSteps() {
			s := builder.Build()
			err = sagaProducer.EmitSaga(l, ctx, s)
			if err != nil {
				l.WithError(err).Errorf("Error emitting restore item saga for quest [%d] item [%d] character [%d].", c.Body.QuestId, c.Body.ItemId, c.CharacterId)
				return
			}
			l.Infof("Restored item [%d] (qty: %d) for quest [%d] character [%d].", foundItem.Id, foundItem.Count, c.Body.QuestId, c.CharacterId)
		}
	}
}
