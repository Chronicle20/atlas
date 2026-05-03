// services/atlas-data/atlas.com/data/skill/effect/point.go
package effect

// PointRestModel is the JSON shape for an LT/RB rectangle corner on a
// skill effect. Pointer-typed on the parent struct so absent rectangles
// serialize as JSON null.
type PointRestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// Point is the in-memory shape for an LT/RB rectangle corner.
type Point struct {
	X int16
	Y int16
}
