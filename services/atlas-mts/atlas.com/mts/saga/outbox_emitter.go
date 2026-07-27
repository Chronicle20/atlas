package saga

import (
	"context"

	msgsaga "atlas-mts/kafka/message/saga"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// OutboxEmitter is a SagaEmitter that persists the saga command as a
// transactional-outbox row bound to tx instead of writing it to Kafka directly,
// so the command publishes iff the enclosing transaction commits.
//
// It exists for the MTS command handlers whose local DB write, status event, AND
// a cross-service escrow saga must be atomic (task-114). listing.PlaceBid and
// listing.Cancel fire their escrow-hold/-release sagas AFTER their own DB write;
// once those methods run inside an outer ExecuteTransaction (so the status event
// can be enqueued atomically), a direct saga emit would fire BEFORE the outer
// commit — a rolled-back bid/cancel would then orphan an escrow move against it
// (the money-losing direction). Routing the saga through the outbox instead keeps
// it in lockstep with the commit. Mirrors the atlas-quest NewOutboxEventEmitter
// pattern for tx-coupled saga commands.
//
// The published message is byte-identical to the direct path: same topic token
// (EnvCommandTopic) and CreateCommandProvider payload, with span+tenant headers
// re-attached from ctx by the drainer at publish time.
type OutboxEmitter struct {
	l   logrus.FieldLogger
	ctx context.Context
	tx  *gorm.DB
}

// NewOutboxEmitter returns a SagaEmitter bound to tx. Construct it inside the
// ExecuteTransaction closure and inject it via listing.WithSagaEmitter so the
// processor's saga emits enqueue onto the same transaction as its writes.
func NewOutboxEmitter(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) *OutboxEmitter {
	return &OutboxEmitter{l: l, ctx: ctx, tx: tx}
}

// Create enqueues the saga command as an outbox row on tx; the drainer publishes
// it to EnvCommandTopic after the transaction commits.
func (e *OutboxEmitter) Create(s Saga) error {
	return outbox.EmitProvider(e.l, e.ctx, e.tx)(msgsaga.EnvCommandTopic)(CreateCommandProvider(s))
}
