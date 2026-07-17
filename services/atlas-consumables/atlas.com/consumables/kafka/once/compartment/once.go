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

// CreationFailedValidator matches the CREATION_FAILED event atlas-inventory
// emits on the compartment status topic when a CREATE_ASSET fails (e.g.
// inventory full). It is the failure half of the reward-grant confirmation; the
// success half (asset CREATED) arrives on the asset status topic instead — see
// kafka/once/asset.CreationValidator. Matching CREATED here would be dead: the
// compartment status topic only emits CREATED for compartment creation.
func CreationFailedValidator(transactionId uuid.UUID) message.Validator[compartment.StatusEvent[compartment.CreateResultEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.CreateResultEventBody]) bool {
		return e.TransactionId == transactionId && e.Type == compartment.StatusEventTypeCreationFailed
	}
}
