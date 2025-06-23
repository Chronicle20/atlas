package transport

import (
	"time"

	"github.com/google/uuid"
)

// LoadSampleRoutes returns sample transport routes and shared vessels
func LoadSampleRoutes() ([]Model, []SharedVesselModel) {
	// Create sample routes
	// Ellinia to Orbis Ferry
	reof := NewBuilder("Ellinia Ferry", 101000300, 200090000, 200090100, 200000100).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute).
		Build()

	// Orbis to Ellinia Ferry
	roef := NewBuilder("Orbis Ferry", 200000100, 200090010, 200090110, 101000300).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute).
		Build()

	// Ludibrium to Orbis Train
	rlot := NewBuilder("Ludibrium Train", 220000100, 220000111, 200000122, 200000100).
		SetBoardingWindowDuration(4 * time.Minute).
		SetPreDepartureDuration(1 * time.Minute).
		SetTravelDuration(15 * time.Minute).
		SetCycleInterval(40 * time.Minute).
		Build()

	// Orbis to Ludibrium Train
	rolt := NewBuilder("Orbis Train", 200000100, 200000121, 200000122, 220000100).
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
