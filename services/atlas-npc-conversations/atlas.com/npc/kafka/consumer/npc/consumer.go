package npc

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/conversation/item"
	consumer2 "atlas-npc-conversations/kafka/consumer"
	npc2 "atlas-npc-conversations/kafka/message/npc"
	producer2 "atlas-npc-conversations/kafka/producer"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// emitConversationStatus is indirected through a package var so consumer
// tests can observe emissions without a broker (same seam shape as
// EmitNpcShopExit in atlas-saga-orchestrator's saga/producer.go).
var emitConversationStatus = func(l logrus.FieldLogger, ctx context.Context, p model.Provider[[]kafka.Message]) error {
	return producer.ProviderImpl(l)(ctx)(npc2.EnvStatusEventTopic)(p)
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_command")(npc2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(npc2.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartConversationCommand(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleContinueConversationCommand(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEndConversationCommand(db)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartItemConversationCommand(db)))); err != nil {
			return err
		}
		return nil
	}
}

func handleStartConversationCommand(db *gorm.DB) message.Handler[npc2.Command[npc2.CommandConversationStartBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandConversationStartBody]) {
		if c.Type != npc2.CommandTypeStartConversation {
			return
		}
		f := field.NewBuilder(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()
		err := conversation.NewProcessor(l, ctx, db).Start(f, c.NpcId, c.CharacterId, c.Body.AccountId)

		// uuid.Nil = the ordinary NPC-talk path, which has no saga awaiting it
		// and must stay exactly as it was. Only a saga-driven start reports.
		if c.TransactionId == uuid.Nil {
			return
		}
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"transaction_id":  c.TransactionId.String(),
				"character_id":    c.CharacterId,
				"npc_template_id": c.NpcId,
			}).Warn("Unable to start saga-driven NPC conversation.")
			emitNpcStartError(l, ctx, c, npc2.StartErrorInternal)
			return
		}
		emitNpcStarted(l, ctx, c)
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

func handleStartItemConversationCommand(db *gorm.DB) message.Handler[npc2.Command[npc2.CommandItemConversationStartBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandItemConversationStartBody]) {
		if c.Type != npc2.CommandTypeStartItemConversation {
			return
		}

		fields := logrus.Fields{
			"transaction_id":  c.TransactionId.String(),
			"character_id":    c.CharacterId,
			"item_id":         c.Body.ItemId,
			"npc_template_id": c.NpcId,
		}

		// The item's own dialogue, keyed by item id. A missing conversation is a
		// content gap, not a fault: START_ERROR fails the awaiting step, the
		// following destroy never runs, and the player keeps the item.
		m, err := item.NewProcessor(l, ctx, db).ByItemIdProvider(c.Body.ItemId)()
		if err != nil {
			l.WithError(err).WithFields(fields).Warn("No conversation authored for scripted item; not consuming.")
			emitItemStartError(l, ctx, c, npc2.StartErrorNoConversationAuthored)
			return
		}

		f := field.NewBuilder(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()

		err = conversation.NewProcessor(l, ctx, db).StartItem(f, c.Body.ItemId, c.NpcId, c.CharacterId, c.Body.AccountId, m.ScriptName(), c.TransactionId, m)
		switch {
		case err == nil:
			l.WithFields(fields).Info("Scripted item conversation started.")
			emitItemStarted(l, ctx, c)
		case errors.Is(err, conversation.ErrAlreadyStartedByThisTransaction):
			// Kafka redelivery of this very command. The dialogue is already
			// open; re-emit success so the awaiting step completes.
			l.WithFields(fields).Debug("Redelivered start command; re-emitting STARTED.")
			emitItemStarted(l, ctx, c)
		case errors.Is(err, conversation.ErrConversationInProgress):
			l.WithFields(fields).Warn("Character is already in a conversation; not consuming.")
			emitItemStartError(l, ctx, c, npc2.StartErrorConversationInProgress)
		default:
			l.WithError(err).WithFields(fields).Error("Unable to start scripted item conversation.")
			emitItemStartError(l, ctx, c, npc2.StartErrorInternal)
		}
	}
}

// emitItemStarted / emitItemStartError are no-ops for a non-saga command
// (uuid.Nil), which is how the ordinary NPC-talk path stays unchanged. Source
// id for an item conversation is the item id, per
// StatusEventStartedBody.SourceId's doc comment.
func emitItemStarted(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandItemConversationStartBody]) {
	if c.TransactionId == uuid.Nil {
		return
	}
	_ = emitConversationStatus(l, ctx, producer2.StartedStatusProvider(c.TransactionId, c.CharacterId, c.NpcId, c.Body.ItemId))
}

func emitItemStartError(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandItemConversationStartBody], reason string) {
	if c.TransactionId == uuid.Nil {
		return
	}
	_ = emitConversationStatus(l, ctx, producer2.StartErrorStatusProvider(c.TransactionId, c.CharacterId, c.NpcId, c.Body.ItemId, reason))
}

// emitNpcStarted / emitNpcStartError are the START_CONVERSATION counterparts.
// Source id for an NPC conversation is the NPC template id, per
// StatusEventStartedBody.SourceId's doc comment.
func emitNpcStarted(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandConversationStartBody]) {
	if c.TransactionId == uuid.Nil {
		return
	}
	_ = emitConversationStatus(l, ctx, producer2.StartedStatusProvider(c.TransactionId, c.CharacterId, c.NpcId, c.NpcId))
}

func emitNpcStartError(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandConversationStartBody], reason string) {
	if c.TransactionId == uuid.Nil {
		return
	}
	_ = emitConversationStatus(l, ctx, producer2.StartErrorStatusProvider(c.TransactionId, c.CharacterId, c.NpcId, c.NpcId, reason))
}
