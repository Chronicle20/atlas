package chalkboard

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Processor is the chalkboard REST client used by the mini-game validation
// ladder. HasOpen reports whether the character has an open chalkboard.
type Processor interface {
	HasOpen(characterId uint32) (bool, error)
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

// HasOpen fetches the character's chalkboard; a 404 (or any fetch failure)
// means there is no open chalkboard, so the check does not block the command.
func (p *ProcessorImpl) HasOpen(characterId uint32) (bool, error) {
	_, err := requestById(characterId)(p.l, p.ctx)
	if err != nil {
		return false, nil
	}
	return true, nil
}
