package buff

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetByCharacterId(characterId uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetByCharacterId drains every page of the character's buffs (the
// upstream list is paginated, task-117).
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(characterBuffsUrl(characterId), 250, Extract, model.Filters[Model]())()
}
