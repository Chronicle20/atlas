package field

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Processor is the interface for field-scoped operations against
// atlas-maps.
type Processor interface {
	// ResetField clears a field's objects and restores its spawn points via
	// atlas-maps' POST .../reset -- Cosmic's MapleMap.resetPQ(difficulty)
	// (task-290 G5).
	ResetField(f field.Model, difficulty int) error
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

func (p *ProcessorImpl) ResetField(f field.Model, difficulty int) error {
	_, err := requestResetField(p.ctx, f, difficulty)(p.l, p.ctx)
	if err != nil {
		return fmt.Errorf("failed to reset field: %w", err)
	}
	return nil
}
