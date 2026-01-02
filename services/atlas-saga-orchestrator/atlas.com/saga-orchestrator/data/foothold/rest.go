package foothold

// PositionInputRestModel represents the input for foothold below lookup.
type PositionInputRestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
}

func (r PositionInputRestModel) GetName() string {
	return "positions"
}

func (r PositionInputRestModel) GetID() string {
	return "0"
}

// PointRestModel represents a 2D point coordinate.
type PointRestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// FootholdRestModel represents a foothold returned from atlas-data.
type FootholdRestModel struct {
	Id     uint32          `json:"id"`
	First  *PointRestModel `json:"first,omitempty"`
	Second *PointRestModel `json:"second,omitempty"`
}

func (r FootholdRestModel) GetName() string {
	return "footholds"
}

func (r FootholdRestModel) GetID() string {
	return "0"
}
