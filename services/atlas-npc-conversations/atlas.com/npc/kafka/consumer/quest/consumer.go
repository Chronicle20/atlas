package quest

import (
	"atlas-npc-conversations/conversation"
	quest2 "atlas-npc-conversations/conversation/quest"
	consumer2 "atlas-npc-conversations/kafka/consumer"
	questMsg "atlas-npc-conversations/kafka/message/quest"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
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
			rf(consumer2.NewConfig(l)("quest_conversation_command")(questMsg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(questMsg.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStartQuestConversationCommand(db))))
	}
}

func handleStartQuestConversationCommand(db *gorm.DB) message.Handler[questMsg.Command[questMsg.StartQuestConversationCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c questMsg.Command[questMsg.StartQuestConversationCommandBody]) {
		if c.Type != questMsg.CommandTypeStartQuestConversation {
			return
		}

		l.Debugf("Received start quest conversation command for quest [%d] character [%d] NPC [%d].", c.QuestId, c.CharacterId, c.NpcId)

		// Get the appropriate state machine based on quest status
		questProcessor := quest2.NewProcessor(l, ctx, db)
		stateMachine, err := questProcessor.GetStateMachineForCharacter(c.QuestId, c.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Failed to get state machine for quest [%d] character [%d].", c.QuestId, c.CharacterId)
			return
		}

		// Build the field model
		f := field.NewBuilder(world.Id(c.Body.WorldId), channel.Id(c.Body.ChannelId), _map.Id(c.Body.MapId)).Build()

		// Start the quest conversation with the appropriate state machine
		err = conversation.NewProcessor(l, ctx, db).StartQuest(f, c.QuestId, c.NpcId, c.CharacterId, &stateMachine)
		if err != nil {
			l.WithError(err).Errorf("Failed to start quest [%d] conversation for character [%d] with NPC [%d].", c.QuestId, c.CharacterId, c.NpcId)
		}
	}
}
