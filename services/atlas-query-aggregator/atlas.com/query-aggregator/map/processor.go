package _map

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

var _ Processor = (*ProcessorImpl)(nil)

// GetPlayerCountInMap retrieves the player count for a single map
// Returns 0 on error to allow graceful degradation
func (p *ProcessorImpl) GetPlayerCountInMap(f field.Model) (int, error) {
	url, err := charactersInMapUrl(p.ctx, f)
	if err != nil {
		p.l.WithError(err).Warnf("Failed to resolve characters-in-map URL for map [%d], using count 0", f.MapId())
		return 0, nil
	}
	resp, err := requests.DrainProvider[RestModel, RestModel](p.l, p.ctx)(url, 250, Extract, model.Filters[RestModel]())()
	if err != nil {
		p.l.WithError(err).Warnf("Failed to get characters in map [%d], using count 0", f.MapId())
		return 0, nil
	}
	count := len(resp)
	p.l.Debugf("Map [%d] has %d players", f.MapId(), count)
	return count, nil
}
