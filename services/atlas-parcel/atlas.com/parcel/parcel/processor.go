package parcel

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// Processor is the parcel custody state machine: creation, mailbox reads,
// and the receive/discard transitions. task-15 adds a processor_custody.go
// in this same package (AcceptCustody/ReleaseCustody/RestoreCustody/
// RemoveCustody) additively — nothing here needs to change to support it.
type Processor interface {
	GetById(id uuid.UUID) (Model, error)
	GetForRecipient(recipientId uint32, worldId world.Id) ([]Model, error)
	GetPendingForSender(senderId uint32) ([]Model, error)
	HasInFlight(characterId uint32) (bool, error)
	Create(m Model) (Model, error)
	Receive(id uuid.UUID, recipientId uint32) (Model, error)
	Discard(id uuid.UUID, recipientId uint32) (Model, error)
}

// ProcessorImpl is the production Processor. now defaults to time.Now and is
// overridden only via withClock (tests) — the state machine never calls
// time.Now directly so ReceivableAt/ExpiresAt/ResolvedAt comparisons are
// deterministic under test.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	now func() time.Time
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		now: time.Now,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// withClock returns a copy of the processor with now replaced — unexported,
// mirroring pending_change's withTransferEligibilityGates seam. Only this
// package's own tests use it; production always gets time.Now via
// NewProcessor.
func (p *ProcessorImpl) withClock(now func() time.Time) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  p.db,
		now: now,
	}
}

// GetById retrieves a single parcel by id.
func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return ById(id)(p.db.WithContext(p.ctx))()
}

// GetForRecipient returns the recipient's pending parcels in a world — the
// mailbox listing (includes parcels not yet receivable, shown with a
// countdown by the client).
func (p *ProcessorImpl) GetForRecipient(recipientId uint32, worldId world.Id) ([]Model, error) {
	return ByRecipient(recipientId, worldId, StatusPending)(p.db.WithContext(p.ctx))()
}

// GetPendingForSender returns the parcels a sender has in flight (still
// pending at the recipient's end).
func (p *ProcessorImpl) GetPendingForSender(senderId uint32) ([]Model, error) {
	return BySender(senderId, StatusPending)(p.db.WithContext(p.ctx))()
}

// HasInFlight reports whether characterId has a parcel it is still
// responsible for: an outbound parcel it sent that is still pending (in
// flight from the instant it is sent), OR an inbound parcel addressed to it
// that has already become receivable. An inbound parcel that has not yet
// become receivable is not the character's problem yet — this asymmetry is
// intentional (design §9.1, gate 12).
func (p *ProcessorImpl) HasInFlight(characterId uint32) (bool, error) {
	now := p.now()
	tdb := p.db.WithContext(p.ctx)

	outbound, err := BySender(characterId, StatusPending)(tdb)()
	if err != nil {
		return false, err
	}
	if len(outbound) > 0 {
		return true, nil
	}

	inbound, err := ReceivableByRecipient(characterId, world.Id(0), now)(tdb)()
	if err != nil {
		return false, err
	}
	return len(inbound) > 0, nil
}

// Create persists a new parcel.
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return Create(p.db.WithContext(p.ctx))(m)
}

// Receive claims a pending, receivable parcel on behalf of recipientId,
// transitioning it to StatusReceived and stamping ResolvedAt with the
// processor's clock. Re-reads the row inside the transaction so a replayed
// receive is award-once (NFR-3): the second delivery finds status != pending
// and returns ErrNotPending.
func (p *ProcessorImpl) Receive(id uuid.UUID, recipientId uint32) (Model, error) {
	return p.resolve(id, recipientId, StatusReceived, func(m Model, now time.Time) error {
		if !m.Receivable(now) {
			if m.Status() != StatusPending {
				return ErrNotPending
			}
			return ErrNotYetReceivable
		}
		return nil
	})
}

// Discard removes a pending parcel on behalf of recipientId, transitioning
// it to StatusDiscarded and stamping ResolvedAt. Unlike Receive, a discard
// does not require ReceivableAt to have passed.
func (p *ProcessorImpl) Discard(id uuid.UUID, recipientId uint32) (Model, error) {
	return p.resolve(id, recipientId, StatusDiscarded, func(m Model, _ time.Time) error {
		if m.Status() != StatusPending {
			return ErrNotPending
		}
		return nil
	})
}

// resolve is the shared race-safe transition underpinning Receive and
// Discard: in one ExecuteTransaction it re-reads the row, validates
// ownership then the caller-supplied gate, and — only once both pass —
// updates status and resolved_at atomically.
func (p *ProcessorImpl) resolve(id uuid.UUID, recipientId uint32, status string, gate func(m Model, now time.Time) error) (Model, error) {
	now := p.now()
	var result Model
	terr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		m, err := ById(id)(tx)()
		if err != nil {
			return ErrNotFound
		}
		if m.RecipientId() != recipientId {
			return ErrNotRecipient
		}
		if err := gate(m, now); err != nil {
			return err
		}
		if err := UpdateStatus(tx)(id, status, now); err != nil {
			return err
		}
		result, err = ById(id)(tx)()
		return err
	})
	if terr != nil {
		return Model{}, terr
	}
	return result, nil
}
