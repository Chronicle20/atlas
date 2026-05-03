package effect

// PointRestModel is the JSON shape for an LT/RB rectangle corner on a
// skill effect. Pointer-typed on the parent struct so absent rectangles
// serialize as JSON null.
type PointRestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}
