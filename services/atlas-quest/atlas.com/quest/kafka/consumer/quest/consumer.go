package quest

import (
	consumer2 "atlas-quest/kafka/consumer"
	quest2 "atlas-quest/kafka/message/quest"
	"atlas-quest/quest"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/world"
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
		}
	}
}

func handleStartQuestCommand(db *gorm.DB) message.Handler[quest2.Command[quest2.StartCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c quest2.Command[quest2.StartCommandBody]) {
		if c.Type != quest2.CommandTypeStart {
			return
		}
		// Kafka commands skip validation by default (validation happens at the caller)
		f := field.NewBuilder(world.Id(c.WorldId), channel.Id(c.ChannelId), 0).Build()
		_, _, err := quest.NewProcessor(l, ctx, db).Start(c.CharacterId, c.Body.QuestId, f, true)
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
		f := field.NewBuilder(world.Id(c.WorldId), channel.Id(c.ChannelId), 0).Build()
		nextQuestId, err := processor.Complete(c.CharacterId, c.Body.QuestId, f, c.Body.Force)
		if err != nil {
			l.WithError(err).Errorf("Error completing quest [%d] for character [%d].", c.Body.QuestId, c.CharacterId)
			return
		}

		// Handle quest chain - auto-start next quest if present
		if nextQuestId > 0 {
			l.Infof("Quest chain detected: starting next quest [%d] for character [%d].", nextQuestId, c.CharacterId)
			_, err = processor.StartChained(c.CharacterId, nextQuestId, f)
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
		err := quest.NewProcessor(l, ctx, db).Forfeit(c.CharacterId, c.Body.QuestId)
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
		err := quest.NewProcessor(l, ctx, db).SetProgress(c.CharacterId, c.Body.QuestId, c.Body.InfoNumber, c.Body.Progress)
		if err != nil {
			l.WithError(err).Errorf("Error updating progress for quest [%d] for character [%d].", c.Body.QuestId, c.CharacterId)
		}
	}
}
