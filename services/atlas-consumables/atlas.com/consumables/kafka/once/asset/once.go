package asset

import (
	"atlas-consumables/kafka/message/asset"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// GrantConfirmedValidator matches the asset event atlas-inventory emits on
// EVENT_TOPIC_ASSET_STATUS when a reward CREATE_ASSET succeeds. A grant can
// land two ways: a fresh stack (CREATED) or a merge into an existing stack the
// player already holds (QUANTITY_CHANGED) — both are success. The compartment
// status topic never emits either for an asset, so this is the only success
// signal. Keyed by transactionId AND the rolled item's templateId: the box's
// own asset events (reserve, consume) share the transactionId but carry the
// box's templateId, so the templateId guard keeps this handler from firing on
// them.
func GrantConfirmedValidator(transactionId uuid.UUID, itemId uint32) message.Validator[asset.StatusEvent[asset.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.CreatedStatusEventBody]) bool {
		return e.TransactionId == transactionId && e.TemplateId == itemId &&
			(e.Type == asset.StatusEventTypeCreated || e.Type == asset.StatusEventTypeQuantityChanged)
	}
}
