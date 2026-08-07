package instance

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestTransformRoute_PopulatesSecondsFields(t *testing.T) {
	route, err := NewRouteBuilder("seconds-route").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(5).
		SetBoardingWindow(90 * time.Second).
		SetTravelDuration(3 * time.Minute).
		Build()
	require.NoError(t, err)

	rm, err := TransformRoute(route)
	require.NoError(t, err)

	assert.Equal(t, uint32(90), rm.BoardingWindowSeconds)
	assert.Equal(t, uint32(180), rm.TravelDurationSeconds)
	// Legacy nanosecond fields are untouched.
	assert.Equal(t, 90*time.Second, rm.BoardingWindow)
	assert.Equal(t, 3*time.Minute, rm.TravelDuration)
}

func TestTransformInstanceStatus_ExposesCreatedAt(t *testing.T) {
	boardingUntil := time.Date(2026, 8, 6, 12, 0, 30, 0, time.UTC)
	arrivalAt := boardingUntil.Add(30 * time.Second)
	inst := NewTransportInstance(uuid.New(), uuid.New(), uuid.New(), boardingUntil, arrivalAt)

	rm, err := TransformInstanceStatus(inst)
	require.NoError(t, err)

	assert.Equal(t, inst.CreatedAt().Format(time.RFC3339), rm.CreatedAt)
	assert.NotEmpty(t, rm.CreatedAt)
}
