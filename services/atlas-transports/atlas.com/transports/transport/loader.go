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
	reof := NewBuilder("Ellinia Ferry", _map.VictoriaRoadElliniaStationId, _map.VictoriaRoadBeforeTakeoffToOrbisId, _map.DuringTheRideToOrbisId, _map.OrbisOrbisStationEnterenceId).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute).
		Build()

	// Orbis to Ellinia Ferry
	roef := NewBuilder("Orbis Ferry", _map.OrbisOrbisStationEnterenceId, _map.OrbisBeforeTakeoffToElliniaId, _map.DuringTheRideToElliniaId, _map.VictoriaRoadElliniaStationId).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute).
		Build()

	// Ludibrium to Orbis Train
	rlot := NewBuilder("Ludibrium Train", _map.LudibriumLudibriumTicketingPlaceId, _map.LudibriumBeforeTheDepartureOrbisId, _map.OrbisBeforeTheDepartureLudibriumId, _map.OrbisOrbisStationEnterenceId).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute).
		Build()

	// Orbis to Ludibrium Train
	rolt := NewBuilder("Orbis Train", _map.OrbisOrbisStationEnterenceId, _map.OrbisStationLudibriumId, _map.OrbisBeforeTheDepartureLudibriumId, _map.LudibriumLudibriumTicketingPlaceId).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute).
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
