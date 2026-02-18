package party_quest

import (
	consumer2 "atlas-party-quests/kafka/consumer"
	"atlas-party-quests/instance"
	pq "atlas-party-quests/kafka/message/party_quest"
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
			rf(consumer2.NewConfig(l)("party_quest_command")(pq.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(pq.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRegisterCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStartCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStageClearAttemptCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStageAdvanceCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleForfeitCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLeaveCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdateStageStateCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdateCustomDataCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleBroadcastMessageCommand(db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterBonusCommand(db))))
	}
}

func handleRegisterCommand(db *gorm.DB) message.Handler[pq.Command[pq.RegisterCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.RegisterCommandBody]) {
		if c.Type != pq.CommandTypeRegister {
			return
		}

		l.Debugf("Handling REGISTER command from character [%d] for quest [%s], party [%d].",
			c.CharacterId, c.Body.QuestId, c.Body.PartyId)

		characters := []instance.CharacterEntry{
			instance.NewCharacterEntry(c.CharacterId, c.WorldId, c.Body.ChannelId),
		}

		_, _ = instance.NewProcessor(l, ctx, db).RegisterAndEmit(c.Body.QuestId, c.Body.PartyId, c.Body.ChannelId, c.Body.MapId, characters)
	}
}

func handleStartCommand(db *gorm.DB) message.Handler[pq.Command[pq.StartCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.StartCommandBody]) {
		if c.Type != pq.CommandTypeStart {
			return
		}

		l.Debugf("Handling START command for instance [%s].", c.Body.InstanceId)
		_ = instance.NewProcessor(l, ctx, db).StartAndEmit(c.Body.InstanceId)
	}
}

func handleStageClearAttemptCommand(db *gorm.DB) message.Handler[pq.Command[pq.StageClearAttemptCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.StageClearAttemptCommandBody]) {
		if c.Type != pq.CommandTypeStageClearAttempt {
			return
		}

		l.Debugf("Handling STAGE_CLEAR_ATTEMPT command for instance [%s].", c.Body.InstanceId)
		_ = instance.NewProcessor(l, ctx, db).StageClearAttemptAndEmit(c.Body.InstanceId)
	}
}

func handleStageAdvanceCommand(db *gorm.DB) message.Handler[pq.Command[pq.StageAdvanceCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.StageAdvanceCommandBody]) {
		if c.Type != pq.CommandTypeStageAdvance {
			return
		}

		l.Debugf("Handling STAGE_ADVANCE command for instance [%s].", c.Body.InstanceId)
		_ = instance.NewProcessor(l, ctx, db).ForceStageCompleteAndEmit(c.Body.InstanceId)
	}
}

func handleForfeitCommand(db *gorm.DB) message.Handler[pq.Command[pq.ForfeitCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.ForfeitCommandBody]) {
		if c.Type != pq.CommandTypeForfeit {
			return
		}

		l.Debugf("Handling FORFEIT command for instance [%s].", c.Body.InstanceId)
		_ = instance.NewProcessor(l, ctx, db).ForfeitAndEmit(c.Body.InstanceId)
	}
}

func handleLeaveCommand(db *gorm.DB) message.Handler[pq.Command[pq.LeaveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.LeaveCommandBody]) {
		if c.Type != pq.CommandTypeLeave {
			return
		}

		l.Debugf("Handling LEAVE command from character [%d].", c.CharacterId)
		_ = instance.NewProcessor(l, ctx, db).LeaveAndEmit(c.CharacterId, "voluntary")
	}
}

func handleUpdateStageStateCommand(db *gorm.DB) message.Handler[pq.Command[pq.UpdateStageStateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.UpdateStageStateCommandBody]) {
		if c.Type != pq.CommandTypeUpdateStageState {
			return
		}

		l.Debugf("Handling UPDATE_STAGE_STATE command for instance [%s].", c.Body.InstanceId)
		_ = instance.NewProcessor(l, ctx, db).UpdateStageState(c.Body.InstanceId, c.Body.ItemCounts, c.Body.MonsterKills)
	}
}

func handleUpdateCustomDataCommand(db *gorm.DB) message.Handler[pq.Command[pq.UpdateCustomDataCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.UpdateCustomDataCommandBody]) {
		if c.Type != pq.CommandTypeUpdateCustomData {
			return
		}

		l.Debugf("Handling UPDATE_CUSTOM_DATA command for instance [%s].", c.Body.InstanceId)
		_ = instance.NewProcessor(l, ctx, db).UpdateCustomData(c.Body.InstanceId, c.Body.Updates, c.Body.Increments)
	}
}

func handleBroadcastMessageCommand(db *gorm.DB) message.Handler[pq.Command[pq.BroadcastMessageCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.BroadcastMessageCommandBody]) {
		if c.Type != pq.CommandTypeBroadcastMessage {
			return
		}

		l.Debugf("Handling BROADCAST_MESSAGE command for instance [%s].", c.Body.InstanceId)
		_ = instance.NewProcessor(l, ctx, db).BroadcastMessageAndEmit(c.Body.InstanceId, c.Body.MessageType, c.Body.Message)
	}
}

func handleEnterBonusCommand(db *gorm.DB) message.Handler[pq.Command[pq.EnterBonusCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pq.Command[pq.EnterBonusCommandBody]) {
		if c.Type != pq.CommandTypeEnterBonus {
			return
		}

		l.Debugf("Handling ENTER_BONUS command for instance [%s].", c.Body.InstanceId)
		_ = instance.NewProcessor(l, ctx, db).EnterBonusAndEmit(c.Body.InstanceId)
	}
}
