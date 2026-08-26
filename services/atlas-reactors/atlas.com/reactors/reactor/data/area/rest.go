package area

import "atlas-reactors/reactor/data/point"

type RestModel struct {
	TL point.RestModel `json:"tl"`
	BR point.RestModel `json:"br"`
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
