package map_

import (
	"math"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type Model struct {
	clock       bool
	returnMapId _map.Id
	fieldLimit  uint32
	town        bool
	footholds   map[uint32]Foothold
}

func (m Model) Clock() bool {
	return m.clock
}

func (m Model) ReturnMapId() _map.Id {
	return m.returnMapId
}

func (m Model) FieldLimit() uint32 {
	return m.fieldLimit
}

func (m Model) Town() bool {
	return m.town
}

func (m Model) NoExpLossOnDeath() bool {
	return _map.NoExpLossOnDeath(m.fieldLimit)
}

// Foothold represents a single foothold segment from the map's foothold tree.
// First and Second are the segment endpoints in MS coordinate space (positive
// y = down). A flat foothold has First.Y == Second.Y; a wall has
// First.X == Second.X.
type Foothold struct {
	Id     uint32
	FirstX int16
	FirstY int16
	SecondX int16
	SecondY int16
}

// SurfaceYOnFoothold returns the surface y for the given foothold id at the
// given x, plus an ok flag. Returns false if the foothold is not present, is
// a wall, or x is outside the foothold's horizontal span.
//
// Mirrors atlas-data/map/processor.go::calcYOnFoothold (slope linear interp).
func (m Model) SurfaceYOnFoothold(fhId uint32, x int16) (int16, bool) {
	fh, ok := m.footholds[fhId]
	if !ok {
		return 0, false
	}
	if fh.FirstX == fh.SecondX { // wall
		return 0, false
	}
	if x < fh.FirstX || x > fh.SecondX {
		return 0, false
	}
	if fh.FirstY == fh.SecondY {
		return fh.FirstY, true
	}
	s1 := math.Abs(float64(fh.SecondY - fh.FirstY))
	s2 := math.Abs(float64(fh.SecondX - fh.FirstX))
	s4 := math.Abs(float64(x - fh.FirstX))
	alpha := math.Atan(s2 / s1)
	beta := math.Atan(s1 / s2)
	s5 := math.Cos(alpha) * (s4 / math.Cos(beta))
	if fh.SecondY < fh.FirstY {
		return fh.FirstY - int16(s5), true
	}
	return fh.FirstY + int16(s5), true
}
