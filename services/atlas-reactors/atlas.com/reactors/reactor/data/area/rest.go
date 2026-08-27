package area

import "atlas-reactors/reactor/data/point"

type RestModel struct {
	TL point.RestModel `json:"tl"`
	BR point.RestModel `json:"br"`
}

// Transform converts a Model into a RestModel. It is the inverse of Extract.
func Transform(m Model) (RestModel, error) {
	tl, err := point.Transform(m.tl)
	if err != nil {
		return RestModel{}, err
	}
	br, err := point.Transform(m.br)
	if err != nil {
		return RestModel{}, err
	}
	return RestModel{
		TL: tl,
		BR: br,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	tl, err := point.Extract(rm.TL)
	if err != nil {
		return Model{}, err
	}
	br, err := point.Extract(rm.BR)
	if err != nil {
		return Model{}, err
	}
	return Model{
		tl: tl,
		br: br,
	}, nil
}
