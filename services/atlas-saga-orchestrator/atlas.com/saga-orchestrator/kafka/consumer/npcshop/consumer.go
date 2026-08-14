package npcshop

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	npcshop "atlas-saga-orchestrator/kafka/message/npcshop"
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
			rf(consumer2.NewConfig(l)("npc_shop_status_event")(npcshop.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(npcshop.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnteredEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterErrorEvent))); err != nil {
			return err
		}
		return nil
	}
}

// handleEnteredEvent completes a pending open_npc_shop step. The ordinary
// NPC-talk path produces ENTER with uuid.Nil, so most events reaching this
// handler match no saga — the nil check and AcceptEvent both decline them.
//
// A redelivered ENTERED for an already-completed step is also declined by
// AcceptEvent, which is the idempotency guarantee the at-least-once topic needs
// (task-221 NFR Idempotency).
func handleEnteredEvent(l logrus.FieldLogger, ctx context.Context, e npcshop.StatusEvent[npcshop.StatusEventEnteredBody]) {
	if e.Type != npcshop.StatusEventTypeEntered {
		return
	}
	if e.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNpcShopEntered); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":  e.TransactionId.String(),
		"character_id":    e.CharacterId,
		"npc_template_id": e.Body.NpcTemplateId,
	}).Debug("NPC shop entered; completing open_npc_shop step.")

	_ = p.StepCompleted(e.TransactionId, true)
}

// handleEnterErrorEvent fails the step, which fails the saga, which means the
// following destroy_asset_from_slot never runs — the cash item survives
// (task-221 FR-4.4).
func handleEnterErrorEvent(l logrus.FieldLogger, ctx context.Context, e npcshop.StatusEvent[npcshop.StatusEventEnterErrorBody]) {
	if e.Type != npcshop.StatusEventTypeEnterError {
		return
	}
	if e.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNpcShopError); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":  e.TransactionId.String(),
		"character_id":    e.CharacterId,
		"npc_template_id": e.Body.NpcTemplateId,
		"reason":          e.Body.Reason,
	}).Error("NPC shop enter failed; failing open_npc_shop step.")

	_ = p.StepCompleted(e.TransactionId, false)
}
