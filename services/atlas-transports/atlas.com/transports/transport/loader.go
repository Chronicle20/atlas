package transport

import (
	"time"

	"github.com/google/uuid"
)

// LoadSampleRoutes returns sample transport routes and shared vessels
func LoadSampleRoutes() ([]Model, []SharedVesselModel) {
	// Create sample routes
	routes := []Model{
		// Ellinia to Orbis Ferry
		NewBuilder().
			SetId("ellinia_to_orbis").
			SetName("Ellinia Ferry").
			SetStartMapID(101000300).
			SetStagingMapID(200090000).
			SetEnRouteMapID(200090100).
			SetDestinationMapID(200000100).
			SetBoardingWindowDuration(1 * time.Minute).
			SetPreDepartureDuration(10 * time.Second).
			SetTravelDuration(90 * time.Second).
			SetCycleInterval(10 * time.Minute).
			Build(),

		// Orbis to Ellinia Ferry
		NewBuilder().
			SetId("orbis_to_ellinia").
			SetName("Orbis Ferry").
			SetStartMapID(200000100).
			SetStagingMapID(200090010).
			SetEnRouteMapID(200090110).
			SetDestinationMapID(101000300).
			SetBoardingWindowDuration(1 * time.Minute).
			SetPreDepartureDuration(10 * time.Second).
			SetTravelDuration(90 * time.Second).
			SetCycleInterval(10 * time.Minute).
			Build(),

		// Ludibrium to Orbis Train
		NewBuilder().
			SetId("ludibrium_to_orbis").
			SetName("Ludibrium Train").
			SetStartMapID(220000100).
			SetStagingMapID(220000111).
			SetEnRouteMapID(200000122).
			SetDestinationMapID(200000100).
			SetBoardingWindowDuration(2 * time.Minute).
			SetPreDepartureDuration(30 * time.Second).
			SetTravelDuration(2 * time.Minute).
			SetCycleInterval(15 * time.Minute).
			Build(),

		// Orbis to Ludibrium Train
		NewBuilder().
			SetId("orbis_to_ludibrium").
			SetName("Orbis Train").
			SetStartMapID(200000100).
			SetStagingMapID(200000121).
			SetEnRouteMapID(200000122).
			SetDestinationMapID(220000100).
			SetBoardingWindowDuration(2 * time.Minute).
			SetPreDepartureDuration(30 * time.Second).
			SetTravelDuration(2 * time.Minute).
			SetCycleInterval(15 * time.Minute).
			Build(),
	}

	// Create shared vessels
	sharedVessels := []SharedVesselModel{
		// Ellinia-Orbis Ferry (shared vessel)
		NewSharedVesselBuilder().
			SetId(uuid.New().String()).
			SetRouteAID("ellinia_to_orbis").
			SetRouteBID("orbis_to_ellinia").
			SetTurnaroundDelay(2 * time.Minute).
			Build(),

		// Ludibrium-Orbis Train (shared vessel)
		NewSharedVesselBuilder().
			SetId(uuid.New().String()).
			SetRouteAID("ludibrium_to_orbis").
			SetRouteBID("orbis_to_ludibrium").
			SetTurnaroundDelay(3 * time.Minute).
			Build(),
	}

	return routes, sharedVessels
}