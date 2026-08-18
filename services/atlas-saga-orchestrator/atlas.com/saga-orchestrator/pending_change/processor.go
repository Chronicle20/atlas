package pending_change

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Processor is the orchestrator-side REST client for the two atlas-character
// endpoints the world-transfer saga (task-227) drives directly rather than
// through Kafka: the dedicated world-change route (Task 7) and the generic
// pending-change resolve route.
type Processor interface {
	// ChangeWorld moves characterId to newWorldId, threading transactionId
	// through so the resulting WORLD_CHANGED status event correlates back to
	// this saga's step (see WorldChangeInputRestModel doc comment).
	ChangeWorld(transactionId uuid.UUID, characterId uint32, newWorldId world.Id) error
	// Resolve moves the pending-change record id to status (APPLIED or
	// REJECTED), carrying reason for the record's audit trail.
	Resolve(characterId uint32, id uuid.UUID, status string, reason string) error
	// CheckTransferEligibility runs atlas-character's read-only gate table
	// (Task 11) for characterId transferring to destinationWorldId. A false
	// eligible carries the gate's rejection reason, which the caller must
	// surface as the step's (and hence the saga's REJECTED record's) failure
	// reason rather than swallow.
	CheckTransferEligibility(characterId uint32, destinationWorldId world.Id) (eligible bool, reason string, err error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ChangeWorld(transactionId uuid.UUID, characterId uint32, newWorldId world.Id) error {
	p.l.Debugf("Changing character [%d] world to [%d].", characterId, newWorldId)
	_, err := changeWorldRequest(characterId, WorldChangeInputRestModel{
		NewWorldId:    newWorldId,
		TransactionId: transactionId,
	})(p.l, p.ctx)
	return err
}

func (p *ProcessorImpl) Resolve(characterId uint32, id uuid.UUID, status string, reason string) error {
	p.l.Debugf("Resolving pending change [%s] for character [%d] to [%s].", id, characterId, status)
	_, err := resolveRequest(characterId, id.String(), ResolveInputRestModel{
		Status: status,
		Reason: reason,
	})(p.l, p.ctx)
	return err
}

func (p *ProcessorImpl) CheckTransferEligibility(characterId uint32, destinationWorldId world.Id) (bool, string, error) {
	resp, err := transferEligibilityRequest(characterId, destinationWorldId)(p.l, p.ctx)
	if err != nil {
		return false, "", err
	}
	return resp.Eligible, resp.Reason, nil
}
