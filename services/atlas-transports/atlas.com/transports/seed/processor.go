package seed

import (
	"atlas-transports/transport"
	"context"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for seed operations
type Processor interface {
	Seed() (SeedResult, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	tp  transport.Processor
}

// NewProcessor creates a new seed processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		tp:  transport.NewProcessor(l, ctx),
	}
}

// Seed loads route configurations from JSON files and seeds them into the transport registry
func (p *ProcessorImpl) Seed() (SeedResult, error) {
	p.l.Infof("Seeding routes for tenant [%s]", p.t.Id())

	result := SeedResult{}

	// Clear existing routes for this tenant
	deletedCount := p.tp.ClearTenant()
	result.DeletedRoutes = deletedCount
	p.l.Debugf("Cleared [%d] existing routes for tenant [%s]", deletedCount, p.t.Id())

	// Load JSON files from the filesystem
	jsonModels, loadErrors := LoadRouteFiles()

	// Track load errors
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Convert JSON models to domain models
	routes := make([]transport.Model, 0, len(jsonModels))
	for _, jm := range jsonModels {
		route, err := ConvertToModel(jm)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.FailedCount++
			continue
		}
		routes = append(routes, route)
	}

	// Add routes to the registry (this also computes schedules)
	if len(routes) > 0 {
		err := p.tp.AddTenant(routes, []transport.SharedVesselModel{})
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.FailedCount = len(routes)
		} else {
			result.CreatedRoutes = len(routes)
		}
	}

	p.l.Infof("Seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		p.t.Id(), result.DeletedRoutes, result.CreatedRoutes, result.FailedCount)

	return result, nil
}

// ConvertToModel converts a JSONModel to a transport.Model
func ConvertToModel(jm JSONModel) (transport.Model, error) {
	// Convert en-route map IDs
	enRouteMapIds := make([]_map.Id, len(jm.EnRouteMapIds))
	for i, id := range jm.EnRouteMapIds {
		enRouteMapIds[i] = _map.Id(id)
	}

	return transport.NewBuilder(jm.Name).
		SetStartMapId(_map.Id(jm.StartMapId)).
		SetStagingMapId(_map.Id(jm.StagingMapId)).
		SetEnRouteMapIds(enRouteMapIds).
		SetDestinationMapId(_map.Id(jm.DestinationMapId)).
		SetObservationMapId(_map.Id(jm.ObservationMapId)).
		SetBoardingWindowDuration(time.Duration(jm.BoardingWindowDurationMinutes) * time.Minute).
		SetPreDepartureDuration(time.Duration(jm.PreDepartureDurationMinutes) * time.Minute).
		SetTravelDuration(time.Duration(jm.TravelDurationMinutes) * time.Minute).
		SetCycleInterval(time.Duration(jm.CycleIntervalMinutes) * time.Minute).
		Build()
}
