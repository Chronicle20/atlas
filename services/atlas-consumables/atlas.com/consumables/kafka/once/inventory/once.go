package inventory

import (
	kafka "atlas-consumables/kafka/message/inventory"
	"context"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func ReservationValidator(transactionId uuid.UUID) message.Validator[kafka.InventoryChangedEvent[kafka.InventoryChangedItemReserveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e kafka.InventoryChangedEvent[kafka.InventoryChangedItemReserveBody]) bool {
		return e.Body.TransactionId == transactionId
	}
}
