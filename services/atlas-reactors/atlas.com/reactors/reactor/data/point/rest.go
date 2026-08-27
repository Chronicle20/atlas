package point

type RestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		x: rm.X,
		y: rm.Y,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		X: m.x,
		Y: m.y,
	}, nil
}
