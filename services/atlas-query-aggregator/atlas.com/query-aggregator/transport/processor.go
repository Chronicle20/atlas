package transport

import (
	"context"
	"fmt"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// Processor provides operations for querying transport routes
type Processor interface {
	GetRouteByStartMap(mapId _map.Id) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new transport processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetRouteByStartMap retrieves a route by its start map ID
// Returns error if no routes found or REST client fails
func (p *ProcessorImpl) GetRouteByStartMap(mapId _map.Id) (Model, error) {
	// Call REST client (returns array of routes)
	resp, err := requestRoutesByStartMap(mapId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Warnf("Failed to get route for start map %d", mapId)
		return Model{}, fmt.Errorf("failed to get route for start map %d: %w", mapId, err)
	}

	// Handle empty array (no matching routes)
	if len(resp) == 0 {
		p.l.Debugf("No routes found for start map %d", mapId)
		return Model{}, fmt.Errorf("no routes found for start map %d", mapId)
	}

	// Warn if multiple routes found (documents assumption: one route per start map)
	if len(resp) > 1 {
		p.l.Warnf("Multiple routes (%d) found for start map %d, using first", len(resp), mapId)
	}

	// Extract first route and transform to domain model
	route, err := Extract(resp[0])
	if err != nil {
		p.l.WithError(err).Errorf("Failed to extract route for start map %d", mapId)
		return Model{}, fmt.Errorf("failed to extract route for start map %d: %w", mapId, err)
	}

	p.l.Debugf("Retrieved route [%s] with state [%s] for start map %d", route.Id(), route.State(), mapId)
	return route, nil
}
