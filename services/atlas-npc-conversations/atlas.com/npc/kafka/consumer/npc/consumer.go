package npc

import (
	"atlas-npc-conversations/conversation"
	consumer2 "atlas-npc-conversations/kafka/consumer"
	npc2 "atlas-npc-conversations/kafka/message/npc"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/map"
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
			rf(consumer2.NewConfig(l)("npc_command")(npc2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(npc2.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStartConversationCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleContinueConversationCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEndConversationCommand(db))))
	}
}

func handleStartConversationCommand(db *gorm.DB) message.Handler[npc2.Command[npc2.CommandConversationStartBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandConversationStartBody]) {
		if c.Type != npc2.CommandTypeStartConversation {
			return
		}
		_ = conversation.NewProcessor(l, ctx, db).Start(field.NewBuilder(world.Id(c.Body.WorldId), channel.Id(c.Body.ChannelId), _map.Id(c.Body.MapId)).Build(), c.NpcId, c.CharacterId, c.Body.AccountId)
	}
}

func handleContinueConversationCommand(db *gorm.DB) message.Handler[npc2.Command[npc2.CommandConversationContinueBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandConversationContinueBody]) {
		if c.Type != npc2.CommandTypeContinueConversation {
			return
		}
		_ = conversation.NewProcessor(l, ctx, db).Continue(c.NpcId, c.CharacterId, c.Body.Action, c.Body.LastMessageType, c.Body.Selection)
	}
}

func handleEndConversationCommand(db *gorm.DB) message.Handler[npc2.Command[npc2.CommandConversationEndBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandConversationEndBody]) {
		if c.Type != npc2.CommandTypeEndConversation {
			return
		}
		_ = conversation.NewProcessor(l, ctx, db).End(c.CharacterId)
	}
}
