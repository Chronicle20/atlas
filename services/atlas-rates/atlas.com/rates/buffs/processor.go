package buffs

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// identity is the no-op transformer for requests.DrainProvider, since
// RestModel is already the target type for this consumer.
func identity(m RestModel) (RestModel, error) {
	return m, nil
}

type Processor interface {
	GetActiveBuffs(characterId uint32) ([]RestModel, error)
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

// GetActiveBuffs retrieves all active buffs for a character from atlas-buffs.
// The upstream list is now paginated (task-117); initializeActiveBuffs (the
// sole caller) must see every buff to compute rate-affecting stat changes,
// so this drains every page rather than fetching just the first.
func (p *ProcessorImpl) GetActiveBuffs(characterId uint32) ([]RestModel, error) {
	return requests.DrainProvider[RestModel, RestModel](p.l, p.ctx)(characterBuffsUrl(characterId), 250, identity, model.Filters[RestModel]())()
}
