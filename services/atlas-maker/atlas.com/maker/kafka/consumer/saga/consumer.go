package saga

import (
	"atlas-maker/craft"
	consumer2 "atlas-maker/kafka/consumer"
	"atlas-maker/kafka/message/saga"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// InitConsumers registers the EVENT_TOPIC_SAGA_STATUS consumer (design §7):
// atlas-maker's craft guard is released by whichever process consumes the
// craft saga's terminal event, and this is that consumer.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
		return func(groupId string) {
			rf(
				consumer2.NewConfig(l)("saga_status_event")(saga.EnvStatusEventTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser),
			)
		}
	}
}

// InitHandlers registers the two terminal handlers -- both COMPLETED and
// FAILED release the in-flight craft guard (design §7): a failed saga that
// never releases reproduces the same craft_in_progress lockout as a
// completed one that never releases.
func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCompleted(l)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFailed(l)))); err != nil {
			return err
		}
		return nil
	}
}

// handleStatusEventCompleted releases the craft guard entry Track-ed under
// e.TransactionId for the message's tenant (TenantHeaderParser puts the
// tenant on ctx before this handler runs).
func handleStatusEventCompleted(l logrus.FieldLogger) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
		if e.Type != saga.StatusEventTypeCompleted {
			return
		}
		t := tenant.MustFromContext(ctx)
		craft.ReleaseInFlightByTransaction(t.Id(), e.TransactionId)
		logger.Debugf("Saga [%s] completed; released in-flight craft guard.", e.TransactionId)
	}
}

// handleStatusEventFailed releases the craft guard entry Track-ed under
// e.TransactionId for the message's tenant. A failed craft saga must release
// exactly like a completed one -- otherwise the character is locked out with
// craft_in_progress until the pod restarts.
func handleStatusEventFailed(l logrus.FieldLogger) message.Handler[saga.StatusEvent[saga.StatusEventFailedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventFailedBody]) {
		if e.Type != saga.StatusEventTypeFailed {
			return
		}
		t := tenant.MustFromContext(ctx)
		craft.ReleaseInFlightByTransaction(t.Id(), e.TransactionId)
		logger.Warnf("Saga [%s] failed. Error: [%s] Reason: [%s] Step: [%s]. Released in-flight craft guard.",
			e.TransactionId, e.Body.ErrorCode, e.Body.Reason, e.Body.FailedStep)
	}
}
