package escrow

import (
	"atlas-trades/kafka/message"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	custodymsg "atlas-trades/kafka/message/custody"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor applies the custody commands and acks them.
//
// Every write is paired with its status event in ONE message.Buffer, so the row
// and the ack land in the same transactional-outbox batch. Splitting them would
// reintroduce exactly the failure this store exists to prevent: a row written
// with no ack leaves the saga step pending until it times out, and an ack with
// no row completes a step whose effect never happened.
type Processor interface {
	Accept(transactionId uuid.UUID, m ItemModel) error
	Release(transactionId uuid.UUID, escrowId uuid.UUID) error
	Restore(transactionId uuid.UUID, escrowId uuid.UUID) error
	Remove(transactionId uuid.UUID, escrowId uuid.UUID) error

	// ClaimItemForReturn is NOT a saga step and emits no ack, for the same
	// reason UpsertMeso does not: it is a plain DB compare-and-set that decides
	// which of the two return paths may submit a trade_unwind for a row (see
	// ClaimItemForReturn), not a custody command the orchestrator is waiting on.
	ClaimItemForReturn(escrowId uuid.UUID) (bool, error)

	// UpsertMeso, DeleteMeso and DeleteResolvedMeso are NOT saga steps and
	// therefore emit no ack: escrowed meso moves through award_mesos, whose own
	// events drive the saga. These only maintain the durable record that makes a
	// refund possible.
	UpsertMeso(roomId uuid.UUID, ownerId character.Id, amount uint32) error
	DeleteMeso(roomId uuid.UUID, ownerId character.Id) error
	DeleteResolvedMeso(roomId uuid.UUID, ownerId character.Id) (bool, error)

	// ArmMesoStake, CommitMesoStake, AbandonMesoStake, and MesoStakeById are
	// likewise NOT saga steps and emit no ack — they exist purely to make an
	// in-flight award_mesos debit durable against room teardown (see
	// MesoEntity's doc comment), not to drive the orchestrator.
	ArmMesoStake(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID, amount uint32, delta int32) error
	CommitMesoStake(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error)
	AbandonMesoStake(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error)
	MesoStakeById(stakeId uuid.UUID) (MesoModel, bool, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) Accept(transactionId uuid.UUID, m ItemModel) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		if err := CreateItem(p.db, p.t)(m); err != nil {
			p.l.WithError(err).Errorf("Unable to write trade escrow row [%s] for character [%d].", m.Id(), m.OwnerId())
			return mb.Put(custodymsg.EnvStatusEventTopic, errorStatusProvider(transactionId, m.Id(), err.Error()))
		}
		return mb.Put(custodymsg.EnvStatusEventTopic, acceptedStatusProvider(transactionId, m.Id()))
	})
}

func (p *ProcessorImpl) Release(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		if err := DeleteItem(p.db, p.t.Id())(escrowId); err != nil {
			p.l.WithError(err).Errorf("Unable to release trade escrow row [%s].", escrowId)
			return mb.Put(custodymsg.EnvStatusEventTopic, errorStatusProvider(transactionId, escrowId, err.Error()))
		}
		return mb.Put(custodymsg.EnvStatusEventTopic, releasedStatusProvider(transactionId, escrowId))
	})
}

func (p *ProcessorImpl) Restore(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		if err := RestoreItem(p.db, p.t.Id())(escrowId); err != nil {
			p.l.WithError(err).Errorf("Unable to restore trade escrow row [%s].", escrowId)
			return mb.Put(custodymsg.EnvStatusEventTopic, errorStatusProvider(transactionId, escrowId, err.Error()))
		}
		return mb.Put(custodymsg.EnvStatusEventTopic, restoredStatusProvider(transactionId, escrowId))
	})
}

// Remove hard-deletes and acks with REMOVED — a distinct type precisely because
// this is a LATE compensating inverse, dispatched after its saga already
// terminated. The orchestrator's consumer deliberately does not route REMOVED
// into StepCompleted; the ack exists for observability, not saga progress.
func (p *ProcessorImpl) Remove(transactionId uuid.UUID, escrowId uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		if err := RemoveItem(p.db, p.t.Id())(escrowId); err != nil {
			p.l.WithError(err).Errorf("Unable to remove trade escrow row [%s].", escrowId)
			return mb.Put(custodymsg.EnvStatusEventTopic, errorStatusProvider(transactionId, escrowId, err.Error()))
		}
		return mb.Put(custodymsg.EnvStatusEventTopic, removedStatusProvider(transactionId, escrowId))
	})
}

func (p *ProcessorImpl) ClaimItemForReturn(escrowId uuid.UUID) (bool, error) {
	return ClaimItemForReturn(p.db, p.t.Id())(escrowId)
}

func (p *ProcessorImpl) UpsertMeso(roomId uuid.UUID, ownerId character.Id, amount uint32) error {
	return UpsertMeso(p.db, p.t)(roomId, ownerId, amount)
}

func (p *ProcessorImpl) DeleteMeso(roomId uuid.UUID, ownerId character.Id) error {
	return DeleteMeso(p.db, p.t.Id())(roomId, ownerId)
}

func (p *ProcessorImpl) DeleteResolvedMeso(roomId uuid.UUID, ownerId character.Id) (bool, error) {
	return DeleteResolvedMeso(p.db, p.t.Id())(roomId, ownerId)
}

func (p *ProcessorImpl) ArmMesoStake(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID, amount uint32, delta int32) error {
	return ArmMesoStake(p.db, p.t)(roomId, ownerId, stakeId, amount, delta)
}

func (p *ProcessorImpl) CommitMesoStake(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
	return CommitMesoStake(p.db, p.t.Id())(roomId, ownerId, stakeId)
}

func (p *ProcessorImpl) AbandonMesoStake(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
	return AbandonMesoStake(p.db, p.t.Id())(roomId, ownerId, stakeId)
}

func (p *ProcessorImpl) MesoStakeById(stakeId uuid.UUID) (MesoModel, bool, error) {
	return MesoStakeById(p.db)(stakeId)
}
