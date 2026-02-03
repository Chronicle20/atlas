package quest

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/kafka/message/quest"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	socketmodel "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("quest_status_event")(quest.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(quest.EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleQuestStarted(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleQuestCompleted(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleQuestForfeited(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleQuestProgressUpdated(sc, wp))))
			}
		}
	}
}

func handleQuestStarted(sc server.Model, wp writer.Producer) message.Handler[quest.StatusEvent[quest.QuestStartedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e quest.StatusEvent[quest.QuestStartedEventBody]) {
		if e.Type != quest.StatusEventTypeStarted {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, announceQuestStarted(l)(ctx)(wp)(e.Body.QuestId, e.Body.Progress))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce quest [%d] started for character [%d].", e.Body.QuestId, e.CharacterId)
		}
	}
}

func announceQuestStarted(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(questId uint32, progress string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(questId uint32, progress string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(questId uint32, progress string) model.Operator[session.Model] {
			return func(questId uint32, progress string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(writer.CharacterStatusMessageOperationUpdateQuestRecordBody(l)(uint16(questId), progress))
			}
		}
	}
}

func handleQuestCompleted(sc server.Model, wp writer.Producer) message.Handler[quest.StatusEvent[quest.QuestCompletedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e quest.StatusEvent[quest.QuestCompletedEventBody]) {
		if e.Type != quest.StatusEventTypeCompleted {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, announceQuestCompleted(l)(ctx)(wp)(e.Body.QuestId, e.Body.CompletedAt, e.Body.Items))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce quest [%d] completed for character [%d].", e.Body.QuestId, e.CharacterId)
		}
	}
}

func announceQuestCompleted(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(questId uint32, completedAt time.Time, items []quest.ItemReward) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(questId uint32, completedAt time.Time, items []quest.ItemReward) model.Operator[session.Model] {
		return func(wp writer.Producer) func(questId uint32, completedAt time.Time, items []quest.ItemReward) model.Operator[session.Model] {
			return func(questId uint32, completedAt time.Time, items []quest.ItemReward) model.Operator[session.Model] {
				return func(s session.Model) error {
					// Send status message to update quest record
					_ = session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(writer.CharacterStatusMessageOperationCompleteQuestRecordBody(l)(uint16(questId), completedAt))(s)

					// Convert items to QuestReward model
					rewards := make([]socketmodel.QuestReward, len(items))
					for i, item := range items {
						rewards[i] = socketmodel.NewQuestReward(item.ItemId, item.Amount)
					}

					// Send quest effect to player showing rewards
					_ = session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterQuestEffectBody(l)("", rewards, 0))(s)

					// Announce quest complete effect to other players in the map
					_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.CharacterEffectForeign)(writer.CharacterQuestEffectForeignBody(l)(s.CharacterId(), "", rewards, 0)))

					return nil
				}
			}
		}
	}
}

func handleQuestForfeited(sc server.Model, wp writer.Producer) message.Handler[quest.StatusEvent[quest.QuestForfeitedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e quest.StatusEvent[quest.QuestForfeitedEventBody]) {
		if e.Type != quest.StatusEventTypeForfeited {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, announceQuestForfeited(l)(ctx)(wp)(e.Body.QuestId))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce quest [%d] forfeited for character [%d].", e.Body.QuestId, e.CharacterId)
		}
	}
}

func announceQuestForfeited(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(questId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(questId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(questId uint32) model.Operator[session.Model] {
			return func(questId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(writer.CharacterStatusMessageOperationForfeitQuestRecordBody(l)(uint16(questId)))
			}
		}
	}
}

func handleQuestProgressUpdated(sc server.Model, wp writer.Producer) message.Handler[quest.StatusEvent[quest.QuestProgressUpdatedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e quest.StatusEvent[quest.QuestProgressUpdatedEventBody]) {
		if e.Type != quest.StatusEventTypeProgressUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, announceQuestProgressUpdated(l)(ctx)(wp)(e.Body.QuestId, e.Body.Progress))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce quest [%d] progress updated for character [%d].", e.Body.QuestId, e.CharacterId)
		}
	}
}

func announceQuestProgressUpdated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(questId uint32, progress string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(questId uint32, progress string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(questId uint32, progress string) model.Operator[session.Model] {
			return func(questId uint32, progress string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(writer.CharacterStatusMessageOperationUpdateQuestRecordBody(l)(uint16(questId), progress))
			}
		}
	}
}
