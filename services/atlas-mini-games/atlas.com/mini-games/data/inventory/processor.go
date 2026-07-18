package inventory

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// Processor is the inventory REST client used by the mini-game validation
// ladder. HasItem reports whether the character owns at least one of itemId.
type Processor interface {
	HasItem(characterId uint32, itemId uint32) (bool, error)
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

// HasItem fetches the compartment that would hold itemId (derived from the item
// classification) with its assets included and reports whether any asset has a
// matching templateId with a non-zero quantity. The item is NOT consumed.
func (p *ProcessorImpl) HasItem(characterId uint32, itemId uint32) (bool, error) {
	it, ok := inventory.TypeFromItemId(item.Id(itemId))
	if !ok {
		return false, nil
	}
	c, err := requestCompartmentByType(characterId, it)(p.l, p.ctx)
	if err != nil {
		return false, err
	}
	for _, a := range c.Assets {
		if a.TemplateId == itemId && a.Quantity > 0 {
			return true, nil
		}
	}
	return false, nil
}
