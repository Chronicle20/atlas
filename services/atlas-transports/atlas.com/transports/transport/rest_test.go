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
