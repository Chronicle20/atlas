package conversation_reward_notice

import (
	consumer2 "atlas-channel/kafka/consumer"
	notice "atlas-channel/kafka/message/conversation_reward_notice"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("conversation_reward_notice")(notice.EnvEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(notice.EnvEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleConversationRewardNotice(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleConversationRewardNotice(sc server.Model, wp writer.Producer) message.Handler[notice.EventBody] {
	return func(l logrus.FieldLogger, ctx context.Context, e notice.EventBody) {
		switch e.Kind {
		case notice.KindItemGain:
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceItemGain(l)(ctx)(wp)(e.ItemId, e.Quantity))
			if err != nil {
				l.WithFields(logrus.Fields{
					"character_id": e.CharacterId,
					"item_id":      e.ItemId,
				}).Info("Skipping conversation item-gain notice — character session not present on this channel.")
			}
		case notice.KindItemLoss:
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceItemLoss(l)(ctx)(wp)(e.ItemId, e.Quantity))
			if err != nil {
				l.WithFields(logrus.Fields{
					"character_id": e.CharacterId,
					"item_id":      e.ItemId,
				}).Info("Skipping conversation item-loss notice — character session not present on this channel.")
			}
		default:
			l.WithField("kind", e.Kind).Debug("Ignoring conversation_reward_notice with unknown kind.")
		}
	}
}

func announceItemGain(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(itemId uint32, quantity uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(itemId uint32, quantity uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(itemId uint32, quantity uint32) model.Operator[session.Model] {
			return func(itemId uint32, quantity uint32) model.Operator[session.Model] {
				rewards := []charcb.QuestReward{{ItemId: itemId, Amount: int32(quantity)}}
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterQuestEffectBody("", rewards, 0))
			}
		}
	}
}

func announceItemLoss(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(itemId uint32, quantity uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(itemId uint32, quantity uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(itemId uint32, quantity uint32) model.Operator[session.Model] {
			return func(itemId uint32, quantity uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageOperationDropLossItemBody(itemId, quantity))
			}
		}
	}
}
