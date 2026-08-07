package instance

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestRouteBuilder_Success(t *testing.T) {
	id := uuid.New()
	route, err := NewRouteBuilder("kerning-square-train").
		SetId(id).
		SetStartMapId(_map.Id(103000000)).
		SetTransitMapIds([]_map.Id{103000100}).
		SetDestinationMapId(_map.Id(103000200)).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, id, route.Id())
	assert.Equal(t, "kerning-square-train", route.Name())
	assert.Equal(t, _map.Id(103000000), route.StartMapId())
	assert.Equal(t, []_map.Id{103000100}, route.TransitMapIds())
	assert.Equal(t, _map.Id(103000200), route.DestinationMapId())
	assert.Equal(t, uint32(6), route.Capacity())
	assert.Equal(t, 10*time.Second, route.BoardingWindow())
	assert.Equal(t, 30*time.Second, route.TravelDuration())
	assert.Equal(t, 80*time.Second, route.MaxLifetime())
}

func TestRouteBuilder_EmptyName(t *testing.T) {
	_, err := NewRouteBuilder("").
		SetTransitMapIds([]_map.Id{100}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestRouteBuilder_ZeroCapacity(t *testing.T) {
	_, err := NewRouteBuilder("test").
		SetTransitMapIds([]_map.Id{100}).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "capacity")
}

func TestRouteBuilder_NoBoardingWindow(t *testing.T) {
	_, err := NewRouteBuilder("test").
		SetTransitMapIds([]_map.Id{100}).
		SetCapacity(6).
		SetTravelDuration(30 * time.Second).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "boarding window")
}

func TestRouteBuilder_ZeroTravelDuration(t *testing.T) {
	route, err := NewRouteBuilder("test").
		SetTransitMapIds([]_map.Id{100}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), route.TravelDuration())
}

func TestRouteBuilder_EmptyTransitMapIds(t *testing.T) {
	_, err := NewRouteBuilder("test").
		SetTransitMapIds([]_map.Id{}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transit map ids")
}

func TestRouteBuilder_GeneratesId(t *testing.T) {
	route, err := NewRouteBuilder("test").
		SetTransitMapIds([]_map.Id{100}).
		SetCapacity(6).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, route.Id())
}

// TestRouteBuilder_EffectFields tables the three effect-field validation
// scenarios: they share the same base route setup and differ only in which
// optional effect field is set and what Build() is expected to do with it.
func TestRouteBuilder_EffectFields(t *testing.T) {
	tests := []struct {
		name string
		// build constructs the route on top of the shared base setup below.
		build func() (RouteModel, error)
		// wantErr, if non-empty, is a substring Build()'s error must contain.
		// If empty, Build() must succeed and check runs against the result.
		wantErr string
		check   func(t *testing.T, route RouteModel)
	}{
		{
			name: "RejectsZeroEffectItemId",
			build: func() (RouteModel, error) {
				return NewRouteBuilder("test").
					SetTransitMapIds([]_map.Id{100}).
					SetCapacity(6).
					SetBoardingWindow(10 * time.Second).
					SetTravelDuration(30 * time.Second).
					SetEffectItemIds([]item.Id{2210016, 0}).
					Build()
			},
			wantErr: "effect item ids",
		},
		{
			// Zero forced-return means "not set", never an error (FR-4.3).
			name: "ZeroForcedReturnMapIdIsNotAnError",
			build: func() (RouteModel, error) {
				return NewRouteBuilder("test").
					SetTransitMapIds([]_map.Id{100}).
					SetCapacity(6).
					SetBoardingWindow(10 * time.Second).
					SetTravelDuration(30 * time.Second).
					SetForcedReturnMapId(_map.Id(0)).
					Build()
			},
			check: func(t *testing.T, route RouteModel) {
				assert.Equal(t, _map.Id(0), route.ForcedReturnMapId())
			},
		},
		{
			// Neither field is required.
			name: "EffectFieldsAreOptional",
			build: func() (RouteModel, error) {
				return NewRouteBuilder("test").
					SetTransitMapIds([]_map.Id{100}).
					SetCapacity(6).
					SetBoardingWindow(10 * time.Second).
					SetTravelDuration(30 * time.Second).
					Build()
			},
			check: func(t *testing.T, route RouteModel) {
				assert.Empty(t, route.EffectItemIds())
				assert.Equal(t, _map.Id(0), route.ForcedReturnMapId())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route, err := tt.build()
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			tt.check(t, route)
		})
	}
}
