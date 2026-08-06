package instance

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RouteRegistry is a Redis-backed atlas.TenantRegistry (route_registry.go:16),
// so EVERY route read in the processor round-trips through
// MarshalJSON/UnmarshalJSON. A field added to the model, the builder and the
// config extractor but NOT to routeModelJSON would be zero at every processor
// call site while passing every other test in this package. This is that guard.
func TestRouteModel_JSONRoundTripPreservesEffectFields(t *testing.T) {
	want, err := NewRouteBuilder("temple-of-time-flight").
		SetStartMapId(_map.Id(240000110)).
		SetTransitMapIds([]_map.Id{200090500, 200090510}).
		SetDestinationMapId(_map.Id(270000100)).
		SetCapacity(1).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(900 * time.Second).
		SetTransitMessage("You are flying towards the Temple of Time.").
		SetEffectItemIds([]item.Id{2210016}).
		SetForcedReturnMapId(_map.Id(240000110)).
		Build()
	assert.NoError(t, err)

	data, err := json.Marshal(want)
	assert.NoError(t, err)

	var got RouteModel
	assert.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, []item.Id{2210016}, got.EffectItemIds())
	assert.Equal(t, _map.Id(240000110), got.ForcedReturnMapId())
	assert.Equal(t, want.Name(), got.Name())
	assert.Equal(t, want.TransitMapIds(), got.TransitMapIds())
}

// A route declaring neither field must survive the round trip with both at
// their zero values — the regression bar for the ten unaffected routes.
func TestRouteModel_JSONRoundTripWithoutEffectFields(t *testing.T) {
	want, err := NewRouteBuilder("ellinia-ereve-ferry").
		SetTransitMapIds([]_map.Id{200090030}).
		SetCapacity(20).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(60 * time.Second).
		Build()
	assert.NoError(t, err)

	data, err := json.Marshal(want)
	assert.NoError(t, err)

	var got RouteModel
	assert.NoError(t, json.Unmarshal(data, &got))

	assert.Empty(t, got.EffectItemIds())
	assert.Equal(t, _map.Id(0), got.ForcedReturnMapId())
}

// EffectItemIds hands out a copy, matching TransitMapIds. Mutating the
// returned slice must not reach back into the model.
func TestRouteModel_EffectItemIdsIsDefensiveCopy(t *testing.T) {
	route, err := NewRouteBuilder("temple-of-time-flight").
		SetTransitMapIds([]_map.Id{200090500}).
		SetCapacity(1).
		SetBoardingWindow(1 * time.Second).
		SetTravelDuration(900 * time.Second).
		SetEffectItemIds([]item.Id{2210016}).
		Build()
	assert.NoError(t, err)

	got := route.EffectItemIds()
	got[0] = item.Id(1)

	assert.Equal(t, []item.Id{2210016}, route.EffectItemIds())
}
