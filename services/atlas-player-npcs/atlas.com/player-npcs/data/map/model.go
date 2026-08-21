package _map

// Rectangle is a bounding box, used for the map's mapArea (FR-4.2 grid
// positioning bounds).
type Rectangle struct {
	x      int16
	y      int16
	width  int16
	height int16
}

func (r Rectangle) X() int16      { return r.x }
func (r Rectangle) Y() int16      { return r.y }
func (r Rectangle) Width() int16  { return r.width }
func (r Rectangle) Height() int16 { return r.height }

// Model is the deploy-time map metadata read from atlas-data.
type Model struct {
	id      uint32
	mapArea *Rectangle
}

func (m Model) Id() uint32          { return m.id }
func (m Model) MapArea() *Rectangle { return m.mapArea }

// GroundPoint is a single query point in a ground-snap request (design
// §5.3, D-2).
type GroundPoint struct {
	x int16
	y int16
}

func NewGroundPoint(x int16, y int16) GroundPoint {
	return GroundPoint{x: x, y: y}
}

func (p GroundPoint) X() int16 { return p.x }
func (p GroundPoint) Y() int16 { return p.y }

// GroundResult is the resolved ground position for one query point.
type GroundResult struct {
	x     int16
	y     int16
	fh    uint32
	found bool
}

func (g GroundResult) X() int16    { return g.x }
func (g GroundResult) Y() int16    { return g.y }
func (g GroundResult) Fh() uint32  { return g.fh }
func (g GroundResult) Found() bool { return g.found }
