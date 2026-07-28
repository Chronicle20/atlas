package foothold

import "math"

// Model is a foothold segment from (x1,y1) to (x2,y2). atlas-data's
// findBelow guarantees x1 <= x2 for the returned segment.
type Model struct {
	id uint32
	x1 int16
	y1 int16
	x2 int16
	y2 int16
}

func (m Model) Id() uint32 { return m.id }

// isWall reports whether the segment is vertical (no floor to land on).
func (m Model) isWall() bool { return m.x1 == m.x2 }

// LandingY returns the y-coordinate on this foothold directly beneath the
// given x. It is a faithful port of atlas-data's calcYOnFoothold
// (services/atlas-data/.../map/model.go) so the drop lands exactly where the
// client expects the floor to be. Returns false if x is off the segment or
// the segment is a wall.
func (m Model) LandingY(x int16) (int16, bool) {
	if m.isWall() {
		return 0, false
	}
	if x < m.x1 || x > m.x2 {
		return 0, false
	}
	if m.y1 == m.y2 {
		return m.y1, true
	}
	s1 := math.Abs(float64(m.y2 - m.y1))
	s2 := math.Abs(float64(m.x2 - m.x1))
	s4 := math.Abs(float64(x - m.x1))
	alpha := math.Atan(s2 / s1)
	beta := math.Atan(s1 / s2)
	s5 := math.Cos(alpha) * (s4 / math.Cos(beta))
	if m.y2 < m.y1 {
		return m.y1 - int16(s5), true
	}
	return m.y1 + int16(s5), true
}
