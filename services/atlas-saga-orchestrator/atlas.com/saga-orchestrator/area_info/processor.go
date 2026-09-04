package area_info

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	Put(characterId uint32, area uint16, info string) error
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

func (p *ProcessorImpl) Put(characterId uint32, area uint16, info string) error {
	_, err := PutAreaInfo(p.l, p.ctx)(characterId, area, info)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to persist area info [%d] for character [%d].", area, characterId)
		return err
	}
	return nil
}
