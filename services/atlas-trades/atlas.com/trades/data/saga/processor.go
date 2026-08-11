package saga

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor reads a saga's outcome from atlas-saga-orchestrator.
type Processor interface {
	// OutcomeProvider yields the outcome of one saga.
	OutcomeProvider(transactionId uuid.UUID) model.Provider[Outcome]
	// Outcome reports what happened to the saga. A non-nil error means the
	// outcome is UNKNOWN — the saga may equally have completed or failed — and
	// callers must leave the settlement unresolved rather than guessing. A
	// not-yet-consumed saga command reads as a 404 here and lands in exactly
	// that branch, which is correct: the trade has not settled yet.
	Outcome(transactionId uuid.UUID) (Outcome, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) OutcomeProvider(transactionId uuid.UUID) model.Provider[Outcome] {
	return requests.Provider[RestModel, Outcome](p.l, p.ctx)(requestSagaById(transactionId), Extract)
}

func (p *ProcessorImpl) Outcome(transactionId uuid.UUID) (Outcome, error) {
	return p.OutcomeProvider(transactionId)()
}
