package transport

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"time"

	"github.com/google/uuid"
)

// LoadSampleRoutes returns sample transport routes and shared vessels
func LoadSampleRoutes() ([]Model, []SharedVesselModel) {
	// Create sample routes
	// Ellinia to Orbis Ferry
	reofBuilder := NewBuilder("Ellinia to Orbis Ferry").
		SetStartMapId(_map.VictoriaRoadElliniaStationId).
		SetStagingMapId(_map.VictoriaRoadBeforeTakeoffToOrbisId).
		AddEnRouteMapId(_map.DuringTheRideToOrbisId).
		AddEnRouteMapId(_map.DuringTheRideCabinToOrbisId).
		SetDestinationMapId(_map.OrbisOrbisStationEnterenceId).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute)

	reof := reofBuilder.Build()

	// Orbis to Ellinia Ferry
	roefBuilder := NewBuilder("Orbis to Ellinia Ferry").
		SetStartMapId(_map.OrbisOrbisStationEnterenceId).
		SetStagingMapId(_map.OrbisBeforeTakeoffToElliniaId).
		AddEnRouteMapId(_map.DuringTheRideToElliniaId).
		AddEnRouteMapId(_map.DuringTheRideCabinToElliniaId).
		SetDestinationMapId(_map.VictoriaRoadElliniaStationId).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute)

	roef := roefBuilder.Build()

	// Ludibrium to Orbis Train
	rlotBuilder := NewBuilder("Ludibrium to Orbis Train").
		SetStartMapId(_map.LudibriumStationOrbisId).
		SetStagingMapId(_map.LudibriumBeforeTheDepartureOrbisId).
		AddEnRouteMapId(_map.OnAVoyageOrbisId).
		SetDestinationMapId(_map.OrbisOrbisStationEnterenceId).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute)

	rlot := rlotBuilder.Build()

	// Orbis to Ludibrium Train
	roltBuilder := NewBuilder("Orbis to Ludibrium Train").
		SetStartMapId(_map.OrbisStationLudibriumId).
		SetStagingMapId(_map.OrbisBeforeTheDepartureLudibriumId).
		AddEnRouteMapId(_map.OnAVoyageLudibriumId).
		SetDestinationMapId(_map.LudibriumLudibriumTicketingPlaceId).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute)

	rolt := roltBuilder.Build()

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
