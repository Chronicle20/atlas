package npcconversation

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	npc "atlas-saga-orchestrator/kafka/message/npc"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_conversation_status_event")(npc.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		var err error
		t, err = topic.EnvProvider(l)(npc.EnvStatusEventTopic)()
		if err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartErrorEvent))); err != nil {
			return err
		}
		return nil
	}
}

// handleStartedEvent completes a pending conversation-start step. The ordinary
// NPC-talk path produces commands with uuid.Nil (not saga-driven), and
// atlas-npc-conversations emits no status event at all for that path, so in
// practice every event reaching this handler should match a saga — but the
// nil check and AcceptEvent both decline any that do not, the same
// declining-by-default posture the npcshop status consumer uses.
//
// A redelivered STARTED for an already-completed step is declined by
// AcceptEvent, which is the idempotency guarantee the at-least-once topic
// needs.
func handleStartedEvent(l logrus.FieldLogger, ctx context.Context, e npc.ConversationStatusEvent[npc.StatusEventStartedBody]) {
	if e.Type != npc.StatusEventTypeStarted {
		return
	}
	if e.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNpcConversationStarted); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":  e.TransactionId.String(),
		"character_id":    e.CharacterId,
		"npc_template_id": e.Body.NpcTemplateId,
		"source_id":       e.Body.SourceId,
	}).Debug("Conversation started; completing conversation-start step.")

	_ = p.StepCompleted(e.TransactionId, true)
}

// handleStartErrorEvent fails the step, which fails the saga, which means the
// following destroy_asset_from_slot (if any) never runs — the player keeps
// the item. reason distinguishes a content gap (NO_CONVERSATION_AUTHORED)
// from a real fault without reading code.
func handleStartErrorEvent(l logrus.FieldLogger, ctx context.Context, e npc.ConversationStatusEvent[npc.StatusEventStartErrorBody]) {
	if e.Type != npc.StatusEventTypeStartError {
		return
	}
	if e.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNpcConversationStartError); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":  e.TransactionId.String(),
		"character_id":    e.CharacterId,
		"npc_template_id": e.Body.NpcTemplateId,
		"source_id":       e.Body.SourceId,
		"reason":          e.Body.Reason,
	}).Warn("Conversation start failed; failing conversation-start step. Item not consumed.")

	_ = p.StepCompleted(e.TransactionId, false)
}
