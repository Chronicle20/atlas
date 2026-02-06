package compartment

import (
	"atlas-consumables/kafka/message/compartment"
	"context"

	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func ReservationValidator(transactionId uuid.UUID, itemId uint32) message.Validator[compartment.StatusEvent[compartment.ReservedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ReservedEventBody]) bool {
		return e.Body.TransactionId == transactionId && e.Body.ItemId == itemId
	}
}
