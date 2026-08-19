package equipslot

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor defines the read surface atlas-channel needs on top of
// atlas-character's equip-slot extensions resource (task-240 task 23, R3) --
// the write side (extending a slot) lives in atlas-cashshop, not here.
type Processor interface {
	GetActive(characterId uint32) ([]RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetActive lists the character's currently-active equip-slot extensions. A
// character with none active returns an empty slice, not an error.
func (p *ProcessorImpl) GetActive(characterId uint32) ([]RestModel, error) {
	return requests.SliceProvider[RestModel, RestModel](p.l, p.ctx)(requestActiveByCharacterId(p.ctx, characterId), Transform, model.Filters[RestModel]())()
}
