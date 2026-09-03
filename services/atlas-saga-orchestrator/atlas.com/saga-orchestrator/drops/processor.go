package drops

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Processor provides drop clearing functionality.
type Processor interface {
	// ClearDrops removes every drop from the given field.
	ClearDrops(f field.Model) error
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

func (p *ProcessorImpl) ClearDrops(f field.Model) error {
	err := requestClearDrops(p.ctx, f)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to clear drops in map %d.", f.MapId())
		return err
	}

	p.l.Debugf("Successfully cleared drops in world %d, channel %d, map %d.", f.WorldId(), f.ChannelId(), f.MapId())
	return nil
}
