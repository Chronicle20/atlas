package compartment

import (
	"atlas-consumables/kafka/message/compartment"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func ReservationValidator(transactionId uuid.UUID, itemId uint32) message.Validator[compartment.StatusEvent[compartment.ReservedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ReservedEventBody]) bool {
		return e.Body.TransactionId == transactionId && e.Body.ItemId == itemId
	}
}

func CreationValidator(transactionId uuid.UUID) message.Validator[compartment.StatusEvent[compartment.CreateResultEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.CreateResultEventBody]) bool {
		return e.TransactionId == transactionId &&
			(e.Type == compartment.StatusEventTypeCreated || e.Type == compartment.StatusEventTypeCreationFailed)
	}
}
