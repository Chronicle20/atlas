package instance

import (
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
