package buffs

import (
	"context"

	"github.com/sirupsen/logrus"
)

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

// GetActiveBuffs retrieves all active buffs for a character from atlas-buffs
func (p *ProcessorImpl) GetActiveBuffs(characterId uint32) ([]RestModel, error) {
	return requestBuffs(characterId)(p.l, p.ctx)
}
