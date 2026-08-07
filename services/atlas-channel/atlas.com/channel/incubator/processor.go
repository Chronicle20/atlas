package incubator

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor defines the interface for incubator reward selection.
type Processor interface {
	// SelectReward rolls one reward for the given Pigmy Egg id via
	// atlas-reward-pools.
	SelectReward(eggId uint32) (Reward, error)
	// SuccessNpcAvailable reports whether the client-side incubator result NPC
	// (SuccessNpcId) is present in the tenant's game data (atlas-data). When
	// false, the client cannot render the incubation result dialog and would
	// crash, so callers must not proceed with the incubation.
	SuccessNpcAvailable() (bool, error)
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
// atlas-reward-pools.
func (p *ProcessorImpl) SelectReward(eggId uint32) (Reward, error) {
	return requests.Provider[RewardRestModel, Reward](p.l, p.ctx)(requestSelectReward(eggId), Extract)()
}

// SuccessNpcAvailable reports whether SuccessNpcId exists in atlas-data. A 404
// (requests.ErrNotFound) means the NPC is absent → (false, nil). Any other
// error is returned so the caller can fail safe (block) rather than risk a
// client crash.
func (p *ProcessorImpl) SuccessNpcAvailable() (bool, error) {
	_, err := requests.Provider[npcRestModel, npcRestModel](p.l, p.ctx)(requestNpcById(SuccessNpcId), func(rm npcRestModel) (npcRestModel, error) { return rm, nil })()
	if errors.Is(err, requests.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
