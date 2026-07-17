package incubator

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for incubator reward selection.
type Processor interface {
	// SelectReward rolls one reward for the given Pigmy Egg id via
	// atlas-gachapons.
	SelectReward(eggId uint32) (Reward, error)
}

// ProcessorImpl implements the Processor interface.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new processor implementation.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// SelectReward rolls one reward for the given Pigmy Egg id via
// atlas-gachapons.
func (p *ProcessorImpl) SelectReward(eggId uint32) (Reward, error) {
	return requests.Provider[RewardRestModel, Reward](p.l, p.ctx)(requestSelectReward(eggId), Extract)()
}
