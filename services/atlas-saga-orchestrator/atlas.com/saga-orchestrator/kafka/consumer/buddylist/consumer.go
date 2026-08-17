package buddylist

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	buddylist2 "atlas-saga-orchestrator/kafka/message/buddylist"
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
			rf(consumer2.NewConfig(l)("buddylist_status_event")(buddylist2.EnvEventTopicBuddyListStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(buddylist2.EnvEventTopicBuddyListStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuddyCapacityChangedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuddyRemovedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleBuddyCapacityChangedEvent(l logrus.FieldLogger, ctx context.Context, e buddylist2.StatusEvent[buddylist2.BuddyCapacityChangeStatusEventBody]) {
	if e.Type != buddylist2.StatusEventTypeBuddyCapacityUpdate {
		return
	}

	// Skip events without a transaction ID (non-saga capacity changes)
	if e.Body.TransactionId == uuid.Nil {
		l.Debugf("Buddy capacity changed event for character [%d] has no transaction ID, skipping saga completion", e.CharacterId)
		return
	}

	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.Body.TransactionId, saga.EventKindBuddyCapacityChanged); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"character_id":   e.CharacterId,
		"new_capacity":   e.Body.Capacity,
	}).Debug("Buddy capacity changed successfully, marking saga step as completed")

	_ = p.StepCompleted(e.Body.TransactionId, true)
}

// handleBuddyRemovedEvent advances a sever_buddies_for_transfer step. N
// buddies means 2N in-flight REQUEST_DELETE commands (one per direction per
// buddy — see handleSeverBuddiesForTransfer in saga/handler.go), so this
// single ack must NOT complete the step on its own: it only reports one of
// the 2N severances landing. saga.AcknowledgeSeverance is the single source
// of truth for "have all of them landed" — StepCompleted is only called once
// it returns true.
func handleBuddyRemovedEvent(l logrus.FieldLogger, ctx context.Context, e buddylist2.StatusEvent[buddylist2.BuddyRemovedStatusEventBody]) {
	if e.Type != buddylist2.StatusEventTypeBuddyRemoved {
		return
	}

	if e.Body.TransactionId == uuid.Nil {
		l.Debugf("Buddy removed event for character [%d] has no transaction ID, skipping saga completion", e.CharacterId)
		return
	}

	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.Body.TransactionId, saga.EventKindBuddyDeleted); !ok {
		return
	}

	if !saga.AcknowledgeSeverance(e.Body.TransactionId, uint32(e.CharacterId), uint32(e.Body.CharacterId)) {
		l.WithFields(logrus.Fields{
			"transaction_id": e.Body.TransactionId.String(),
			"character_id":   e.CharacterId,
			"buddy_id":       e.Body.CharacterId,
		}).Debug("Buddy severance acknowledged; step remains pending other severances.")
		return
	}

	if err := p.StepCompleted(e.Body.TransactionId, true); err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": e.Body.TransactionId.String(),
		}).Error("All buddy severances acknowledged but step completion failed.")
	}
}
