package asset

import (
	"atlas-messages/data/equipable"
	"context"
	"github.com/sirupsen/logrus"
	"math"
)

type Processor interface {
	Exists(itemId uint32) bool
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	sp  equipable.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		sp:  equipable.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) Exists(itemId uint32) bool {
	inventoryType := byte(math.Floor(float64(itemId) / 1000000))
	if inventoryType == 1 {
		_, err := p.sp.GetById(itemId)
		if err != nil {
			return false
		}
		return true
	}

	return true
}
