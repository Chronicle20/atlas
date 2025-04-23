package inventory

import (
	"atlas-messages/character/inventory/item"
	"atlas-messages/equipment/statistics"
	"context"
	"github.com/sirupsen/logrus"
	"math"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	sp  *statistics.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		sp:  statistics.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) Exists(itemId uint32) bool {
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

func (p *Processor) CreateItem(characterId uint32, itemId uint32, quantity uint16) (item.Model, error) {
	rm, err := requestCreateItem(characterId, itemId, quantity)(p.l, p.ctx)
	if err != nil {
		return item.Model{}, err
	}
	return item.Extract(rm)
}
