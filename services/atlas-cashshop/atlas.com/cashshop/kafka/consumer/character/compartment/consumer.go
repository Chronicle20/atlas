package compartment

import (
	"atlas-cashshop/cashshop"
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/message/character/compartment"
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
			rf(consumer2.NewConfig(l)("compartment_status_event")(compartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(compartment.EnvEventTopicStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventError(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCashItemMoved(db))))
		}
	}
}

// handleStatusEventError handles error status events from the compartment service
// If the error code is CASH_ITEM_MOVE_FAILED, it releases the asset reservation and emits a cash shop error event
func handleStatusEventError(db *gorm.DB) message.Handler[compartment.StatusEvent[compartment.ErrorEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ErrorEventBody]) {
		if e.Type != compartment.StatusEventTypeError {
			return
		}

		// Check if the error code is CASH_ITEM_MOVE_FAILED
		if e.Body.ErrorCode != compartment.StatusEventErrorTypeCashItemMoveFailed {
			return
		}

		l.Debugf("Received CASH_ITEM_MOVE_FAILED error for character [%d], cash item [%d].", e.CharacterId, e.Body.CashItemId)

		// Use the character compartment processor to handle the error
		err := cashshop.NewProcessor(l, ctx, db).MoveFromFailedAndEmit(e.CharacterId, e.Body.CashItemId)
		if err != nil {
			l.WithError(err).Errorf("Failed to handle cash item move failed error for character [%d], cash item [%d].", e.CharacterId, e.Body.CashItemId)
			return
		}
	}
}

// handleStatusEventCashItemMoved handles cash item moved status events from the compartment service
// It removes the asset reservation, removes the cash compartment - cash item association, and emits a cash shop status event
func handleStatusEventCashItemMoved(db *gorm.DB) message.Handler[compartment.StatusEvent[compartment.CashItemMovedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.CashItemMovedEventBody]) {
		if e.Type != compartment.StatusEventTypeCashItemMoved {
			return
		}

		l.Debugf("Received CASH_ITEM_MOVED event for character [%d], cash item [%d], slot [%d].", e.CharacterId, e.Body.CashItemId, e.Body.Slot)

		// Use the character compartment processor to handle the cash item moved event
		err := cashshop.NewProcessor(l, ctx, db).MovedFromAndEmit(e.CharacterId, e.Body.CashItemId, e.CompartmentId, e.Body.Slot)
		if err != nil {
			l.WithError(err).Errorf("Failed to handle cash item moved event for character [%d], cash item [%d].", e.CharacterId, e.Body.CashItemId)
			return
		}
	}
}
