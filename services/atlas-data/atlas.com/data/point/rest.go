package point

type RestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		X: m.X(),
		Y: m.Y(),
	}, nil
}
