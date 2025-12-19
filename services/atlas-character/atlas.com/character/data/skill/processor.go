package skill

import (
	"atlas-character/data/skill/effect"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(uniqueId uint32) (Model, error)
	GetEffect(uniqueId uint32, level byte) (effect.Model, error)
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

func (p *ProcessorImpl) GetById(uniqueId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(uniqueId), Extract)()
}

func (p *ProcessorImpl) GetEffect(uniqueId uint32, level byte) (effect.Model, error) {
	s, err := p.GetById(uniqueId)
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
