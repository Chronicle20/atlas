package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func restTestRoute(t *testing.T) Model {
	t.Helper()
	m, err := NewBuilder("Rest Route").
		SetId(uuid.New()).
		SetStartMapId(_map.Id(100)).
		SetStagingMapId(_map.Id(101)).
		SetEnRouteMapIds([]_map.Id{_map.Id(102)}).
		SetDestinationMapId(_map.Id(103)).
		SetObservationMapId(_map.Id(104)).
		SetState(AwaitingReturn).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(15 * time.Minute).
		Build()
	require.NoError(t, err)
	return m
}

func TestTransform_PopulatesSecondsFields(t *testing.T) {
	rm, err := Transform(restTestRoute(t))
	require.NoError(t, err)

	assert.Equal(t, uint32(300), rm.BoardingWindowSeconds)
	assert.Equal(t, uint32(120), rm.PreDepartureSeconds)
	assert.Equal(t, uint32(600), rm.TravelDurationSeconds)
	assert.Equal(t, uint32(900), rm.CycleIntervalSeconds)

	// The legacy nanosecond field is untouched.
	assert.Equal(t, 15*time.Minute, rm.CycleInterval)
}

func TestExtract_RoundTripsDurations(t *testing.T) {
	original := restTestRoute(t)
	rm, err := Transform(original)
	require.NoError(t, err)

	back, err := Extract(rm)
	require.NoError(t, err)

	assert.Equal(t, original.BoardingWindowDuration(), back.BoardingWindowDuration())
	assert.Equal(t, original.PreDepartureDuration(), back.PreDepartureDuration())
	assert.Equal(t, original.TravelDuration(), back.TravelDuration())
	assert.Equal(t, original.CycleInterval(), back.CycleInterval())
}

// pinTimeNow swaps the package clock seam for the duration of a test.
func pinTimeNow(t *testing.T, at time.Time) {
	t.Helper()
	previous := timeNow
	timeNow = func() time.Time { return at }
	t.Cleanup(func() { timeNow = previous })
}

func TestTransform_PopulatesNextTransition(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pinTimeNow(t, now)

	routeId := uuid.New()
	sched := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(h, m int) time.Time {
		return time.Date(sched.Year(), sched.Month(), sched.Day(), h, m, 0, 0, time.UTC)
	}
	trip, err := NewTripScheduleBuilder().
		SetTripId(uuid.New()).
		SetRouteId(routeId).
		SetBoardingOpen(at(12, 30)).
		SetBoardingClosed(at(12, 35)).
		SetDeparture(at(12, 37)).
		SetArrival(at(12, 47)).
		Build()
	require.NoError(t, err)

	m, err := NewBuilder("Next Transition Route").
		SetId(routeId).
		SetStartMapId(_map.Id(100)).
		SetStagingMapId(_map.Id(101)).
		SetEnRouteMapIds([]_map.Id{_map.Id(102)}).
		SetDestinationMapId(_map.Id(103)).
		SetObservationMapId(_map.Id(104)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(15 * time.Minute).
		SetSchedule([]TripScheduleModel{trip}).
		Build()
	require.NoError(t, err)

	rm, err := Transform(m)
	require.NoError(t, err)

	assert.Equal(t, string(OpenEntry), rm.NextState)
	assert.Equal(t, "2026-08-06T12:30:00Z", rm.NextTransitionAt)
	// state is whatever Evaluate decided on the same `now` - they cannot diverge.
	assert.Equal(t, string(AwaitingReturn), rm.State)
}

func TestTransform_OutOfServiceHasNoTransition(t *testing.T) {
	pinTimeNow(t, time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC))

	rm, err := Transform(restTestRoute(t))
	require.NoError(t, err)

	assert.Equal(t, string(OutOfService), rm.State)
	assert.Equal(t, "", rm.NextState)
	assert.Equal(t, "", rm.NextTransitionAt)
}
