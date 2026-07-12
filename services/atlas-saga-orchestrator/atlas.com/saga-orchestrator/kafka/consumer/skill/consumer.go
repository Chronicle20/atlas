package skill

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	skill2 "atlas-saga-orchestrator/kafka/message/skill"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("skill_status_event")(skill2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(skill2.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillCreatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillUpdatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillDeletedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillSpTransferredEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSkillErrorEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleSkillCreatedEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventCreatedBody]) {
	if e.Type != skill2.StatusEventTypeCreated {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindSkillCreated); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleSkillUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventUpdatedBody]) {
	if e.Type != skill2.StatusEventTypeUpdated {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindSkillUpdated); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

// handleSkillDeletedEvent drives StepCompleted(true) when atlas-skills responds
// to a saga-correlated REQUEST_DELETE (plan Phase 5 / Phase 6).
func handleSkillDeletedEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventDeletedBody]) {
	if e.Type != skill2.StatusEventTypeDeleted {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindSkillDeleted); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

// handleSkillSpTransferredEvent completes a point_reset transfer_sp step when
// atlas-skills confirms the SP move (SP Reset, task-126).
func handleSkillSpTransferredEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventSpTransferredBody]) {
	if e.Type != skill2.StatusEventTypeSpTransferred {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindSkillSpTransferred); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

// handleSkillErrorEvent marks a point_reset transfer_sp step failed when
// atlas-skills rejects the TRANSFER_SP command, threading the service's error
// code + detail onto the step result map (Task 14 error-threading contract).
func handleSkillErrorEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventErrorBody]) {
	if e.Type != skill2.StatusEventTypeError {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindSkillSpTransferError); !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_id":   e.CharacterId,
		"error":          e.Body.Error,
		"detail":         e.Body.Detail,
	}).Debug("SP transfer rejected; marking saga step failed.")
	_ = p.StepCompletedWithResult(e.TransactionId, false, map[string]any{"errorCode": e.Body.Error, "errorDetail": e.Body.Detail})
}
