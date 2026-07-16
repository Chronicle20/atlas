package shop

import (
	"testing"

	"atlas-merchant/data/portal"

	"github.com/stretchr/testify/assert"
)

func mkPortal(name string, ptype uint8, x, y int16, target uint32) portal.Model {
	p, _ := portal.Extract(portal.RestModel{Id: "0", Name: name, Type: ptype, X: x, Y: y, TargetMapId: target})
	return p
}

func TestNearBlockingPortal_SpawnPointExcluded(t *testing.T) {
	// The exact Free Market failure: character stands on the spawn point ("sp",
	// type 0, no target). Spawn points do not block placement.
	portals := []portal.Model{mkPortal("sp", 0, 828, -210, 999999999)}
	assert.False(t, nearBlockingPortal(828, -146, portals))
}

func TestNearBlockingPortal_TeleportPortalWithinRange(t *testing.T) {
	// A real teleport portal (type 1, real target) within 120 blocks placement.
	portals := []portal.Model{mkPortal("up00", 1, 828, -200, 910000001)}
	assert.True(t, nearBlockingPortal(828, -146, portals)) // dist 54 < 120
}

func TestNearBlockingPortal_TeleportPortalOutOfRange(t *testing.T) {
	portals := []portal.Model{mkPortal("up00", 1, 676, -421, 910000001)}
	assert.False(t, nearBlockingPortal(828, -146, portals)) // dist ~314 > 120
}

func TestNearBlockingPortal_MapExitPortalIgnored(t *testing.T) {
	// type 2 (map-exit) portals are not teleport portals — they never block.
	portals := []portal.Model{mkPortal("out00", 2, 828, -180, 910000000)}
	assert.False(t, nearBlockingPortal(828, -146, portals))
}

func TestNearBlockingPortal_TeleportNoTargetIgnored(t *testing.T) {
	portals := []portal.Model{mkPortal("up00", 1, 828, -180, 999999999)}
	assert.False(t, nearBlockingPortal(828, -146, portals))
}

func TestNearBlockingPortal_FreeMarketRoomAllowsPlacement(t *testing.T) {
	// Full FM room 910000001 portal set from the live pr-env: standing at
	// (828,-146) must be allowed — only the nearby "sp" spawn point is in range,
	// and it is excluded.
	portals := []portal.Model{
		mkPortal("sp", 0, 828, -210, 999999999),
		mkPortal("up00", 1, -274, 30, 910000001),
		mkPortal("dn00", 1, 676, -421, 910000001),
		mkPortal("out00", 2, 790, 35, 910000000),
	}
	assert.False(t, nearBlockingPortal(828, -146, portals))
}

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
