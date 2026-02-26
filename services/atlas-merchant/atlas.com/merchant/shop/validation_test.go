package shop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFreemarketRoom_ValidRooms(t *testing.T) {
	// Henesys Free Market <1>
	assert.True(t, IsFreemarketRoom(100000111))
	// Perion Free Market <9>
	assert.True(t, IsFreemarketRoom(102000109))
	// El Nath Free Market <3>
	assert.True(t, IsFreemarketRoom(211000113))
	// Ludibrium Free Market <1>
	assert.True(t, IsFreemarketRoom(220000201))
	// Hidden Street Free Market <1>
	assert.True(t, IsFreemarketRoom(910000001))
	// Hidden Street Free Market <22>
	assert.True(t, IsFreemarketRoom(910000022))
}

func TestIsFreemarketRoom_InvalidRooms(t *testing.T) {
	assert.False(t, IsFreemarketRoom(0))
	assert.False(t, IsFreemarketRoom(100000100)) // Henesys but not FM
	assert.False(t, IsFreemarketRoom(910000023)) // One past last hidden street FM
	assert.False(t, IsFreemarketRoom(999999999))
}

func TestManhattanDistance(t *testing.T) {
	assert.Equal(t, 0, manhattanDistance(0, 0, 0, 0))
	assert.Equal(t, 10, manhattanDistance(5, 5, 10, 10))
	assert.Equal(t, 200, manhattanDistance(-100, 0, 100, 0))
	assert.Equal(t, 20, manhattanDistance(10, 10, 0, 0))
	assert.Equal(t, 100, manhattanDistance(-50, -50, 0, 0))
}

func TestIsNearExistingShop_NoShops(t *testing.T) {
	provider := func() ([]Model, error) {
		return []Model{}, nil
	}
	assert.False(t, IsNearExistingShop(100000111, 0, 0, provider))
}

func TestIsNearExistingShop_FarAway(t *testing.T) {
	provider := func() ([]Model, error) {
		return []Model{
			{mapId: 100000111, x: 500, y: 500},
		}, nil
	}
	assert.False(t, IsNearExistingShop(100000111, 0, 0, provider))
}

func TestIsNearExistingShop_TooClose(t *testing.T) {
	provider := func() ([]Model, error) {
		return []Model{
			{mapId: 100000111, x: 10, y: 10},
		}, nil
	}
	assert.True(t, IsNearExistingShop(100000111, 0, 0, provider))
}

func TestIsNearExistingShop_ExactThreshold(t *testing.T) {
	// At exactly 100 distance, should NOT be considered too close (< 100 required)
	provider := func() ([]Model, error) {
		return []Model{
			{mapId: 100000111, x: 100, y: 0},
		}, nil
	}
	assert.False(t, IsNearExistingShop(100000111, 0, 0, provider))
}

func TestIsNearExistingShop_DifferentMap(t *testing.T) {
	provider := func() ([]Model, error) {
		return []Model{
			{mapId: 100000112, x: 0, y: 0}, // Different map
		}, nil
	}
	assert.False(t, IsNearExistingShop(100000111, 0, 0, provider))
}
