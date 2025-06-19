package transport

import (
	"time"

	"github.com/google/uuid"
)

// LoadSampleRoutes returns sample transport routes and shared vessels
func LoadSampleRoutes() ([]Model, []SharedVesselModel) {
	// Create sample routes
	// Ellinia to Orbis Ferry
	reof := NewBuilder().
		SetId(uuid.New()).
		SetName("Ellinia Ferry").
		SetStartMapID(101000300).
		SetStagingMapID(200090000).
		SetEnRouteMapID(200090100).
		SetDestinationMapID(200000100).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(10 * time.Second).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(20 * time.Minute).
		Build()

	// Orbis to Ellinia Ferry
	roef := NewBuilder().
		SetId(uuid.New()).
		SetName("Orbis Ferry").
		SetStartMapID(200000100).
		SetStagingMapID(200090010).
		SetEnRouteMapID(200090110).
		SetDestinationMapID(101000300).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(10 * time.Second).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(20 * time.Minute).
		Build()

	// Ludibrium to Orbis Train
	rlot := NewBuilder().
		SetId(uuid.New()).
		SetName("Ludibrium Train").
		SetStartMapID(220000100).
		SetStagingMapID(220000111).
		SetEnRouteMapID(200000122).
		SetDestinationMapID(200000100).
		SetBoardingWindowDuration(2 * time.Minute).
		SetPreDepartureDuration(30 * time.Second).
		SetTravelDuration(2 * time.Minute).
		SetCycleInterval(15 * time.Minute).
		Build()

	// Orbis to Ludibrium Train
	rolt := NewBuilder().
		SetId(uuid.New()).
		SetName("Orbis Train").
		SetStartMapID(200000100).
		SetStagingMapID(200000121).
		SetEnRouteMapID(200000122).
		SetDestinationMapID(220000100).
		SetBoardingWindowDuration(2 * time.Minute).
		SetPreDepartureDuration(30 * time.Second).
		SetTravelDuration(2 * time.Minute).
		SetCycleInterval(15 * time.Minute).
		Build()

	routes := []Model{reof, roef, rlot, rolt}

	// Create shared vessels
	sharedVessels := []SharedVesselModel{
		// Ellinia-Orbis Ferry (shared vessel)
		NewSharedVesselBuilder().
			SetId(uuid.New().String()).
			SetRouteAID(reof.Id()).
			SetRouteBID(roef.Id()).
			SetTurnaroundDelay(0 * time.Second).
			Build(),

		// Ludibrium-Orbis Train (shared vessel)
		NewSharedVesselBuilder().
			SetId(uuid.New().String()).
			SetRouteAID(rlot.Id()).
			SetRouteBID(rolt.Id()).
			SetTurnaroundDelay(0 * time.Second).
			Build(),
	}

	return routes, sharedVessels
}
