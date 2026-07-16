package incubator

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for incubator-rewards configuration
// operations.
type Processor interface {
	// GetRewards returns the incubator reward pool for the tenant in context.
	GetRewards() ([]Reward, error)
	// GetRewardsForEgg returns the reward pool for one Pigmy Egg (region).
	GetRewardsForEgg(eggId uint32) ([]Reward, error)
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

// GetRewards returns the incubator reward pool for the tenant in context.
func (p *ProcessorImpl) GetRewards() ([]Reward, error) {
	t := tenant.MustFromContext(p.ctx)
	return requests.SliceProvider[RewardRestModel, Reward](p.l, p.ctx)(requestRewards(t.Id().String()), Extract, model.Filters[Reward]())()
}

// GetRewardsForEgg returns the reward pool for one Pigmy Egg (region).
func (p *ProcessorImpl) GetRewardsForEgg(eggId uint32) ([]Reward, error) {
	all, err := p.GetRewards()
	if err != nil {
		return nil, err
	}
	return FilterByEgg(all, eggId), nil
}
