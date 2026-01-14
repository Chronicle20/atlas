package transport

import (
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Build_Validation(t *testing.T) {
	tests := []struct {
		name        string
		builder     func() *Builder
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid route builds successfully",
			builder: func() *Builder {
				return NewBuilder("Test Route").
					SetStartMapId(100).
					SetStagingMapId(101).
					SetEnRouteMapIds([]_map.Id{102}).
					SetDestinationMapId(103).
					SetBoardingWindowDuration(5 * time.Minute).
					SetPreDepartureDuration(2 * time.Minute).
					SetTravelDuration(10 * time.Minute).
					SetCycleInterval(30 * time.Minute)
			},
			expectError: false,
		},
		{
			name: "Empty name fails validation",
			builder: func() *Builder {
				return NewBuilder("").
					SetEnRouteMapIds([]_map.Id{102}).
					SetBoardingWindowDuration(5 * time.Minute).
					SetPreDepartureDuration(2 * time.Minute).
					SetTravelDuration(10 * time.Minute).
					SetCycleInterval(30 * time.Minute)
			},
			expectError: true,
			errorMsg:    "route name must not be empty",
		},
		{
			name: "No en-route map IDs fails validation",
			builder: func() *Builder {
				return NewBuilder("Test Route").
					SetBoardingWindowDuration(5 * time.Minute).
					SetPreDepartureDuration(2 * time.Minute).
					SetTravelDuration(10 * time.Minute).
					SetCycleInterval(30 * time.Minute)
			},
			expectError: true,
			errorMsg:    "at least one en-route map ID is required",
		},
		{
			name: "Zero boarding window duration fails validation",
			builder: func() *Builder {
				return NewBuilder("Test Route").
					SetEnRouteMapIds([]_map.Id{102}).
					SetBoardingWindowDuration(0).
					SetPreDepartureDuration(2 * time.Minute).
					SetTravelDuration(10 * time.Minute).
					SetCycleInterval(30 * time.Minute)
			},
			expectError: true,
			errorMsg:    "boarding window duration must be positive",
		},
		{
			name: "Negative boarding window duration fails validation",
			builder: func() *Builder {
				return NewBuilder("Test Route").
					SetEnRouteMapIds([]_map.Id{102}).
					SetBoardingWindowDuration(-5 * time.Minute).
					SetPreDepartureDuration(2 * time.Minute).
					SetTravelDuration(10 * time.Minute).
					SetCycleInterval(30 * time.Minute)
			},
			expectError: true,
			errorMsg:    "boarding window duration must be positive",
		},
		{
			name: "Zero pre-departure duration fails validation",
			builder: func() *Builder {
				return NewBuilder("Test Route").
					SetEnRouteMapIds([]_map.Id{102}).
					SetBoardingWindowDuration(5 * time.Minute).
					SetPreDepartureDuration(0).
					SetTravelDuration(10 * time.Minute).
					SetCycleInterval(30 * time.Minute)
			},
			expectError: true,
			errorMsg:    "pre-departure duration must be positive",
		},
		{
			name: "Zero travel duration fails validation",
			builder: func() *Builder {
				return NewBuilder("Test Route").
					SetEnRouteMapIds([]_map.Id{102}).
					SetBoardingWindowDuration(5 * time.Minute).
					SetPreDepartureDuration(2 * time.Minute).
					SetTravelDuration(0).
					SetCycleInterval(30 * time.Minute)
			},
			expectError: true,
			errorMsg:    "travel duration must be positive",
		},
		{
			name: "Zero cycle interval fails validation",
			builder: func() *Builder {
				return NewBuilder("Test Route").
					SetEnRouteMapIds([]_map.Id{102}).
					SetBoardingWindowDuration(5 * time.Minute).
					SetPreDepartureDuration(2 * time.Minute).
					SetTravelDuration(10 * time.Minute).
					SetCycleInterval(0)
			},
			expectError: true,
			errorMsg:    "cycle interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.builder().Build()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, Model{}, result)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, Model{}, result)
			}
		})
	}
}

func TestSharedVesselBuilder_Build_Validation(t *testing.T) {
	validRouteA := uuid.New()
	validRouteB := uuid.New()

	tests := []struct {
		name        string
		builder     func() *SharedVesselBuilder
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid shared vessel builds successfully",
			builder: func() *SharedVesselBuilder {
				return NewSharedVesselBuilder().
					SetName("Test Vessel").
					SetRouteAID(validRouteA).
					SetRouteBID(validRouteB).
					SetTurnaroundDelay(5 * time.Minute)
			},
			expectError: false,
		},
		{
			name: "Nil route A ID fails validation",
			builder: func() *SharedVesselBuilder {
				return NewSharedVesselBuilder().
					SetName("Test Vessel").
					SetRouteBID(validRouteB).
					SetTurnaroundDelay(5 * time.Minute)
			},
			expectError: true,
			errorMsg:    "route A ID must not be nil",
		},
		{
			name: "Nil route B ID fails validation",
			builder: func() *SharedVesselBuilder {
				return NewSharedVesselBuilder().
					SetName("Test Vessel").
					SetRouteAID(validRouteA).
					SetTurnaroundDelay(5 * time.Minute)
			},
			expectError: true,
			errorMsg:    "route B ID must not be nil",
		},
		{
			name: "Zero turnaround delay fails validation",
			builder: func() *SharedVesselBuilder {
				return NewSharedVesselBuilder().
					SetName("Test Vessel").
					SetRouteAID(validRouteA).
					SetRouteBID(validRouteB).
					SetTurnaroundDelay(0)
			},
			expectError: true,
			errorMsg:    "turnaround delay must be positive",
		},
		{
			name: "Negative turnaround delay fails validation",
			builder: func() *SharedVesselBuilder {
				return NewSharedVesselBuilder().
					SetName("Test Vessel").
					SetRouteAID(validRouteA).
					SetRouteBID(validRouteB).
					SetTurnaroundDelay(-5 * time.Minute)
			},
			expectError: true,
			errorMsg:    "turnaround delay must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.builder().Build()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, SharedVesselModel{}, result)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, SharedVesselModel{}, result)
			}
		})
	}
}

func TestTripScheduleBuilder_Build_Validation(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	validRouteId := uuid.New()

	tests := []struct {
		name        string
		builder     func() *TripScheduleBuilder
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid trip schedule builds successfully",
			builder: func() *TripScheduleBuilder {
				return NewTripScheduleBuilder().
					SetRouteId(validRouteId).
					SetBoardingOpen(now).
					SetBoardingClosed(now.Add(5 * time.Minute)).
					SetDeparture(now.Add(7 * time.Minute)).
					SetArrival(now.Add(17 * time.Minute))
			},
			expectError: false,
		},
		{
			name: "Nil route ID fails validation",
			builder: func() *TripScheduleBuilder {
				return NewTripScheduleBuilder().
					SetBoardingOpen(now).
					SetBoardingClosed(now.Add(5 * time.Minute)).
					SetDeparture(now.Add(7 * time.Minute)).
					SetArrival(now.Add(17 * time.Minute))
			},
			expectError: true,
			errorMsg:    "route ID must not be nil",
		},
		{
			name: "Boarding open not before boarding closed fails validation",
			builder: func() *TripScheduleBuilder {
				return NewTripScheduleBuilder().
					SetRouteId(validRouteId).
					SetBoardingOpen(now.Add(5 * time.Minute)).
					SetBoardingClosed(now). // Before boarding open
					SetDeparture(now.Add(7 * time.Minute)).
					SetArrival(now.Add(17 * time.Minute))
			},
			expectError: true,
			errorMsg:    "boarding open must be before boarding closed",
		},
		{
			name: "Boarding open equals boarding closed fails validation",
			builder: func() *TripScheduleBuilder {
				return NewTripScheduleBuilder().
					SetRouteId(validRouteId).
					SetBoardingOpen(now).
					SetBoardingClosed(now). // Same as boarding open
					SetDeparture(now.Add(7 * time.Minute)).
					SetArrival(now.Add(17 * time.Minute))
			},
			expectError: true,
			errorMsg:    "boarding open must be before boarding closed",
		},
		{
			name: "Boarding closed not before departure fails validation",
			builder: func() *TripScheduleBuilder {
				return NewTripScheduleBuilder().
					SetRouteId(validRouteId).
					SetBoardingOpen(now).
					SetBoardingClosed(now.Add(10 * time.Minute)).
					SetDeparture(now.Add(5 * time.Minute)). // Before boarding closed
					SetArrival(now.Add(17 * time.Minute))
			},
			expectError: true,
			errorMsg:    "boarding closed must be before departure",
		},
		{
			name: "Departure not before arrival fails validation",
			builder: func() *TripScheduleBuilder {
				return NewTripScheduleBuilder().
					SetRouteId(validRouteId).
					SetBoardingOpen(now).
					SetBoardingClosed(now.Add(5 * time.Minute)).
					SetDeparture(now.Add(20 * time.Minute)).
					SetArrival(now.Add(17 * time.Minute)) // Before departure
			},
			expectError: true,
			errorMsg:    "departure must be before arrival",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.builder().Build()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, TripScheduleModel{}, result)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, TripScheduleModel{}, result)
			}
		})
	}
}

func TestBuilder_ModelFieldsAreSet(t *testing.T) {
	routeId := uuid.New()
	route, err := NewBuilder("Test Route").
		SetId(routeId).
		SetStartMapId(100).
		SetStagingMapId(101).
		SetEnRouteMapIds([]_map.Id{102, 103}).
		SetDestinationMapId(104).
		SetObservationMapId(105).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()

	require.NoError(t, err)

	assert.Equal(t, routeId, route.Id())
	assert.Equal(t, "Test Route", route.Name())
	assert.Equal(t, _map.Id(100), route.StartMapId())
	assert.Equal(t, _map.Id(101), route.StagingMapId())
	assert.Equal(t, []_map.Id{102, 103}, route.EnRouteMapIds())
	assert.Equal(t, _map.Id(104), route.DestinationMapId())
	assert.Equal(t, _map.Id(105), route.ObservationMapId())
	assert.Equal(t, 5*time.Minute, route.BoardingWindowDuration())
	assert.Equal(t, 2*time.Minute, route.PreDepartureDuration())
	assert.Equal(t, 10*time.Minute, route.TravelDuration())
	assert.Equal(t, 30*time.Minute, route.CycleInterval())
	assert.Equal(t, OutOfService, route.State())
}

func TestBuilder_AddEnRouteMapId(t *testing.T) {
	route, err := NewBuilder("Test Route").
		AddEnRouteMapId(102).
		AddEnRouteMapId(103).
		AddEnRouteMapId(104).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()

	require.NoError(t, err)
	assert.Equal(t, []_map.Id{102, 103, 104}, route.EnRouteMapIds())
}
