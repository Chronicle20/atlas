package parcel

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is atlas-channel's read-only view of atlas-parcel's custody
// resource.
type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

// GetForRecipient lists recipientId's pending mailbox in worldId.
func (p *Processor) GetForRecipient(recipientId uint32, worldId world.Id) ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestForRecipient(p.ctx, recipientId, worldId), Extract, model.Filters[Model]())()
}

// CountPending answers the mailbox-capacity check (design §6.2) — the
// length of recipientId's pending mailbox in worldId, the only pre-flight
// besides recipient resolution that leaves the channel.
//
// worldId is required here even though the brief's interface sketch shows
// CountPending(recipientId) alone: atlas-parcel's GET /parcels rejects a
// recipient filter with no world filter (services/atlas-parcel/.../
// resource.go handleGetParcels), and defaulting it internally would be
// exactly the world-0-is-not-a-sentinel mistake this branch has already
// flagged four times elsewhere. The caller already has the recipient's
// resolved world (it filtered ByNameProvider's result to it), so passing it
// through costs nothing.
func (p *Processor) CountPending(recipientId uint32, worldId world.Id) (int, error) {
	ms, err := p.GetForRecipient(recipientId, worldId)
	if err != nil {
		return 0, err
	}
	return len(ms), nil
}

// GetById retrieves a single parcel by id (Task 18's receive/discard arms).
func (p *Processor) GetById(id uuid.UUID) (Model, error) {
	rm, err := requestById(p.ctx, id)(p.l, p.ctx)
	if err != nil {
		return Model{}, err
	}
	return Extract(rm)
}

// Discard marks parcelId discarded on behalf of recipientId (design §4.4 /
// §7.3): discard is deliberately not a saga, so this is a direct
// PATCH /parcels/{id} against atlas-parcel, which owns the recipient/status
// validation server-side (parcel.ProcessorImpl.Discard).
func (p *Processor) Discard(id uuid.UUID, recipientId uint32) (Model, error) {
	rm, err := discardRequest(p.ctx, id, recipientId)(p.l, p.ctx)
	if err != nil {
		return Model{}, err
	}
	return Extract(rm)
}

// MarkNotified stamps LastNotified on id (Task 21's SHOW_PARCEL consumer,
// design §5.3, FR-24): a direct PATCH /parcels/{id}/notify against
// atlas-parcel, mirroring Discard's direct-PATCH shape. Not a saga —
// nothing leaves custody, this is bookkeeping invisible to the player.
func (p *Processor) MarkNotified(id uuid.UUID) error {
	_, err := notifyRequest(p.ctx, id)(p.l, p.ctx)
	return err
}
