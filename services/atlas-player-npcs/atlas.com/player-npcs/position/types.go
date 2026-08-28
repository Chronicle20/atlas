// Package position implements the two FR-4.7 Player NPC positioners: the
// grid lattice used on most Hall of Fame maps, and the podium slot layout
// used on the five ranked maps (routing.IsPodiumMap). Both positioners are
// pure geometry -- ground resolution is injected through SnapFunc so the
// package holds no state and calls no other service.
package position

import "errors"

// Tuning holds the FR-4.7 grid-positioner parameters for one map.
type Tuning struct {
	InitialX     int16
	InitialY     int16
	AreaX        int16
	AreaY        int16
	AreaSteps    byte
	OrganizeArea bool
}

// Rect is an axis-aligned placement footprint in map pixel coordinates.
type Rect struct {
	X, Y, W, H int16
}

// Point is a resolved placement: a horizontal coordinate paired with the
// ground it snapped to (CY, the foothold's y at that x) and the foothold id
// itself.
type Point struct {
	X  int16
	CY int16
	Fh uint32
}

// Placement pairs a script id with its resolved footprint/point and the
// step it was placed at -- the unit Reorganize walks and re-derives.
type Placement struct {
	ScriptId uint32
	Rect     Rect
	Point    Point
	Step     byte
}

// SnapFunc resolves a batch of candidate points to ground (foothold y and
// id) in one round trip. It is injected so the grid/podium positioners stay
// pure and testable without an HTTP client. NextGridPosition issues exactly
// one SnapFunc call per step's lattice walk (design 5.3).
type SnapFunc func(points []Point) ([]Point, error)

// ErrMapFull is returned when no free lattice slot exists at the map's
// bounded areaSteps.
var ErrMapFull = errors.New("map_full")

// ErrInvalidStep is returned when a caller passes step == 0 to a
// positioner that would otherwise divide by zero.
var ErrInvalidStep = errors.New("invalid_step")
