package monster

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Processor interface {
	CreateMonster(f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) error
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

func (p *ProcessorImpl) CreateMonster(f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) error {
	_, err := requestCreate(f, monsterId, x, y, fh, team)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Creating monster for map %s.", f.Id())
	}
	return err
}
