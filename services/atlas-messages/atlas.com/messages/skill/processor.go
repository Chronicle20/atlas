package skill

import (
	"atlas-messages/skill/effect"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) GetById(uniqueId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(uniqueId), Extract)()
}

func (p *Processor) GetEffect(uniqueId uint32, level byte) (effect.Model, error) {
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
