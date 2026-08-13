package instance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
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

// TransformRoute is the layer-4 projection from the domain RouteModel to the
// debug REST resource. None of the package's other tests call it directly,
// so a regression that dropped the EffectItemIds/ForcedReturnMapId field
// mappings (rest.go:54-55) would go undetected without this test.
func TestTransformRoute_MapsEffectFields(t *testing.T) {
	route, err := NewRouteBuilder("temple-of-time-flight").
		SetStartMapId(_map.Id(240000110)).
		SetTransitMapIds([]_map.Id{200090500, 200090510}).
		SetDestinationMapId(_map.Id(270000100)).
		SetCapacity(1).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(900 * time.Second).
		SetEffectItemIds([]item.Id{2210016}).
		SetForcedReturnMapId(_map.Id(240000110)).
		Build()
	assert.NoError(t, err)

	got, err := TransformRoute(route)
	assert.NoError(t, err)

	assert.Equal(t, []item.Id{2210016}, got.EffectItemIds)
	assert.Equal(t, _map.Id(240000110), got.ForcedReturnMapId)
}

// A route declaring neither field must project cleanly — empty/zero, not
// garbage — matching the ten unaffected routes.
func TestTransformRoute_WithoutEffectFieldsProjectsCleanly(t *testing.T) {
	route, err := NewRouteBuilder("ellinia-ereve-ferry").
		SetTransitMapIds([]_map.Id{200090030}).
		SetCapacity(20).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(60 * time.Second).
		Build()
	assert.NoError(t, err)

	got, err := TransformRoute(route)
	assert.NoError(t, err)

	assert.Empty(t, got.EffectItemIds)
	assert.Equal(t, _map.Id(0), got.ForcedReturnMapId)
}

// TransformRoute must not alias RouteModel's internal slice — mutating the
// projected REST model's EffectItemIds must not reach back into the route.
// Mirrors TestRouteModel_EffectItemIdsIsDefensiveCopy (model_json_test.go),
// one layer up the stack.
func TestTransformRoute_EffectItemIdsIsNotAliased(t *testing.T) {
	route, err := NewRouteBuilder("temple-of-time-flight").
		SetTransitMapIds([]_map.Id{200090500}).
		SetCapacity(1).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(900 * time.Second).
		SetEffectItemIds([]item.Id{2210016}).
		Build()
	assert.NoError(t, err)

	got, err := TransformRoute(route)
	assert.NoError(t, err)

	got.EffectItemIds[0] = item.Id(1)

	assert.Equal(t, []item.Id{2210016}, route.EffectItemIds())
}
