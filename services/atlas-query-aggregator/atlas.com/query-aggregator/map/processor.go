package _map

import (
	"context"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

// Processor provides operations for querying map player counts
type Processor interface {
	GetPlayerCountInMap(f field.Model) (int, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new map processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetPlayerCountInMap retrieves the player count for a single map
// Returns 0 on error to allow graceful degradation
func (p *ProcessorImpl) GetPlayerCountInMap(f field.Model) (int, error) {
	resp, err := requestCharactersInMap(f)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Warnf("Failed to get characters in map [%d], using count 0", f.MapId())
		return 0, nil
	}
	count := len(resp)
	p.l.Debugf("Map [%d] has %d players", f.MapId(), count)
	return count, nil
}
