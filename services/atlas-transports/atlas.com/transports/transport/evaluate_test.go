package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// evaluateTestRoute builds a route with a single-trip schedule whose
// boundaries are expressed as offsets from `base`.
func evaluateTestRoute(t *testing.T, routeId uuid.UUID, trips []TripScheduleModel) Model {
	t.Helper()
	m, err := NewBuilder("Evaluate Route").
		SetId(routeId).
		SetStartMapId(_map.Id(100)).
		SetStagingMapId(_map.Id(101)).
		SetEnRouteMapIds([]_map.Id{_map.Id(102)}).
		SetDestinationMapId(_map.Id(103)).
		SetObservationMapId(_map.Id(104)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		SetSchedule(trips).
		Build()
	require.NoError(t, err)
	return m
}

func evaluateTestTrip(t *testing.T, routeId uuid.UUID, open, closed, departure, arrival time.Time) TripScheduleModel {
	t.Helper()
	trip, err := NewTripScheduleBuilder().
		SetTripId(uuid.New()).
		SetRouteId(routeId).
		SetBoardingOpen(open).
		SetBoardingClosed(closed).
		SetDeparture(departure).
		SetArrival(arrival).
		Build()
	require.NoError(t, err)
	return trip
}

func TestEvaluate_Branches(t *testing.T) {
	routeId := uuid.New()
	// The schedule's own date is deliberately NOT the evaluation date -
	// this is the stale-date property Evaluate has to be immune to.
	sched := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	at := func(h, m int) time.Time {
		return time.Date(sched.Year(), sched.Month(), sched.Day(), h, m, 0, 0, time.UTC)
	}
	todayAt := func(h, m int) time.Time {
		return time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, time.UTC)
	}

	// One trip: boards 12:30, closes 12:35, departs 12:37, arrives 12:47.
	trip := func() []TripScheduleModel {
		return []TripScheduleModel{evaluateTestTrip(t, routeId, at(12, 30), at(12, 35), at(12, 37), at(12, 47))}
	}

	tests := []struct {
		name          string
		now           time.Time
		trips         []TripScheduleModel
		expectedState RouteState
		expectedNext  RouteState
		expectedAt    time.Time
	}{
		{
			name:          "No trips is out of service with a zero boundary",
			now:           now,
			trips:         []TripScheduleModel{},
			expectedState: OutOfService,
			expectedNext:  "",
			expectedAt:    time.Time{},
		},
		{
			name:          "Before boarding opens counts down to boarding open",
			now:           todayAt(12, 0),
			trips:         trip(),
			expectedState: AwaitingReturn,
			expectedNext:  OpenEntry,
			expectedAt:    todayAt(12, 30),
		},
		{
			name:          "Boarding open counts down to boarding closed",
			now:           todayAt(12, 31),
			trips:         trip(),
			expectedState: OpenEntry,
			expectedNext:  LockedEntry,
			expectedAt:    todayAt(12, 35),
		},
		{
			name:          "Locked entry counts down to departure",
			now:           todayAt(12, 36),
			trips:         trip(),
			expectedState: LockedEntry,
			expectedNext:  InTransit,
			expectedAt:    todayAt(12, 37),
		},
		{
			name:          "In transit counts down to arrival",
			now:           todayAt(12, 40),
			trips:         trip(),
			expectedState: InTransit,
			expectedNext:  AwaitingReturn,
			expectedAt:    todayAt(12, 47),
		},
		{
			// With only one trip on the schedule, once its arrival has passed there
			// is no in-transit trip and no future trip left to govern the state, so
			// Evaluate falls through to the same OutOfService/zero-boundary result as
			// the no-trips case. This mirrors the pre-existing processStateChange
			// behavior pinned by state_test.go's "After arrival" case, which is the
			// identical single-trip-past-arrival shape and must keep returning
			// OutOfService. Wrapping to a next boarding open only happens when a
			// later trip exists on the same schedule (see the next case) or the
			// governing trip's own boundary crosses midnight (see the
			// "Midnight-crossing trip" case below).
			name:          "Past the last arrival with no other trips is out of service",
			now:           todayAt(13, 0),
			trips:         trip(),
			expectedState: OutOfService,
			expectedNext:  "",
			expectedAt:    time.Time{},
		},
		{
			name: "Past an arrival with a later trip counts down to that trip's boarding open",
			now:  todayAt(13, 0),
			trips: []TripScheduleModel{
				evaluateTestTrip(t, routeId, at(12, 30), at(12, 35), at(12, 37), at(12, 47)),
				evaluateTestTrip(t, routeId, at(14, 0), at(14, 5), at(14, 7), at(14, 17)),
			},
			expectedState: AwaitingReturn,
			expectedNext:  OpenEntry,
			expectedAt:    todayAt(14, 0),
		},
		{
			name: "Midnight-crossing trip in transit counts down to arrival after midnight",
			now:  todayAt(23, 50),
			trips: []TripScheduleModel{
				// Arrival's absolute date is pushed a day past departure's so the
				// builder's departure-before-arrival invariant holds; Evaluate only
				// compares time-of-day, so this does not change what's under test.
				evaluateTestTrip(t, routeId, at(23, 30), at(23, 35), at(23, 40), at(0, 20).Add(24*time.Hour)),
			},
			expectedState: InTransit,
			expectedNext:  AwaitingReturn,
			expectedAt:    todayAt(0, 20).Add(24 * time.Hour),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := evaluateTestRoute(t, routeId, tc.trips)
			got := m.Evaluate(tc.now)
			assert.Equal(t, tc.expectedState, got.State)
			assert.Equal(t, tc.expectedNext, got.NextState)
			assert.True(t, got.NextAt.Equal(tc.expectedAt),
				"NextAt = %s, want %s", got.NextAt, tc.expectedAt)
		})
	}
}

// TestEvaluate_StateMatchesProcessStateChange pins the refactor: the wrapper
// must keep returning exactly what Evaluate decided.
func TestEvaluate_StateMatchesProcessStateChange(t *testing.T) {
	routeId := uuid.New()
	sched := time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC)
	m := evaluateTestRoute(t, routeId, []TripScheduleModel{
		evaluateTestTrip(t, routeId, sched, sched.Add(5*time.Minute), sched.Add(7*time.Minute), sched.Add(17*time.Minute)),
	})
	for _, offset := range []time.Duration{0, 6 * time.Minute, 8 * time.Minute, 20 * time.Minute} {
		now := time.Date(2026, 8, 6, 12, 30, 0, 0, time.UTC).Add(offset)
		assert.Equal(t, m.Evaluate(now).State, m.processStateChange(now))
	}
}

// TestEvaluate_NextAtIsAlwaysInTheFuture guards the materialization rule.
func TestEvaluate_NextAtIsAlwaysInTheFuture(t *testing.T) {
	routeId := uuid.New()
	sched := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	trips := []TripScheduleModel{
		evaluateTestTrip(t, routeId,
			sched.Add(8*time.Hour), sched.Add(8*time.Hour+5*time.Minute),
			sched.Add(8*time.Hour+7*time.Minute), sched.Add(8*time.Hour+17*time.Minute)),
	}
	m := evaluateTestRoute(t, routeId, trips)
	base := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	for minute := 0; minute < 24*60; minute += 7 {
		now := base.Add(time.Duration(minute) * time.Minute)
		got := m.Evaluate(now)
		if got.State == OutOfService {
			continue
		}
		assert.True(t, got.NextAt.After(now), "at %s NextAt = %s is not in the future", now, got.NextAt)
		assert.True(t, got.NextAt.Sub(now) <= 24*time.Hour, "at %s NextAt = %s is more than a day out", now, got.NextAt)
	}
}
