package inventory

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// AccommodationRequest is one item to ask atlas-inventory whether it could grant.
type AccommodationRequest struct {
	ItemId   uint32
	Quantity uint32
}

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[Model]
	GetByCharacterId(characterId uint32) (Model, error)
	// CanAccommodate asks atlas-inventory whether every listed item could
	// currently be granted to the character (each evaluated independently). It
	// is merge-aware: a full tab does not block a stackable that fits an existing
	// stack. Returns true only when all items are accommodatable.
	CanAccommodate(characterId uint32, items []AccommodationRequest) (bool, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) CanAccommodate(characterId uint32, items []AccommodationRequest) (bool, error) {
	if len(items) == 0 {
		return true, nil
	}
	return requests.Provider[accommodationOutputRestModel, bool](p.l, p.ctx)(
		requestCheckAccommodation(characterId, items),
		func(rm accommodationOutputRestModel) (bool, error) { return rm.Accommodated, nil },
	)()
}
