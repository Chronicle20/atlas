package point

import "atlas-data/point"

type RestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

func Transform(m point.Model) (RestModel, error) {
	return RestModel{
		X: m.X(),
		Y: m.Y(),
	}, nil
}
