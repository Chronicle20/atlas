package asset

import (
	"atlas-consumables/kafka/message/asset"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CreationValidator matches the asset CREATED event atlas-inventory emits on
// EVENT_TOPIC_ASSET_STATUS when a CREATE_ASSET succeeds. The reward flow's
// success once-handler keys off this: the compartment status topic never emits
// CREATED for an asset creation (only for compartment creation), so the
// asset-created confirmation is the correct success signal.
func CreationValidator(transactionId uuid.UUID) message.Validator[asset.StatusEvent[asset.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.CreatedStatusEventBody]) bool {
		return e.TransactionId == transactionId && e.Type == asset.StatusEventTypeCreated
	}
}
