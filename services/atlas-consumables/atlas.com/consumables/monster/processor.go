package monster

import (
	"context"

	"github.com/Chronicle20/atlas-constants/field"
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

func (p *Processor) CreateMonster(f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) error {
	_, err := requestCreate(f, monsterId, x, y, fh, team)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Creating monster for map %s.", f.Id())
	}
	return err
}
