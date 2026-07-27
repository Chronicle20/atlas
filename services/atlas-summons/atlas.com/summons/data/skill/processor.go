package skill

import (
	"atlas-summons/data/skill/effect"
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(skillId uint32) (Model, error)
	GetEffect(skillId uint32, level byte) (effect.Model, error)
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

func (p *ProcessorImpl) GetById(skillId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(skillId), Extract)()
}

func (p *ProcessorImpl) GetEffect(skillId uint32, level byte) (effect.Model, error) {
	s, err := p.GetById(skillId)
	if err != nil {
		return effect.Model{}, err
	}
	if level == 0 {
		return effect.Model{}, nil
	}
	if len(s.Effects()) < int(level-1) {
		return effect.Model{}, errors.New("level out of bounds")
	}
	return s.Effects()[level-1], nil
}
