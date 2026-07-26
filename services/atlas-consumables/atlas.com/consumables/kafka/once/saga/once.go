package saga

import (
	"atlas-consumables/kafka/message/saga"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
)

// TransactionValidator passes for the terminal (COMPLETED or FAILED) status
// event of one saga transaction. Used with message.OneTimeConfig so a single
// registration observes whichever outcome arrives (a saga emits exactly one).
func TransactionValidator(transactionId uuid.UUID) message.Validator[saga.StatusEvent[json.RawMessage]] {
	return func(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[json.RawMessage]) bool {
		return e.TransactionId == transactionId &&
			(e.Type == saga.StatusEventTypeCompleted || e.Type == saga.StatusEventTypeFailed)
	}
}
