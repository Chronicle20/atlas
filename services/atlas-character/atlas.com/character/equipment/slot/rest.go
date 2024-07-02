package slot

import (
	"atlas-character/equipable"
)

type RestModel struct {
	Position  Position             `json:"position"`
	Equipable *equipable.RestModel `json:"equipable"`
}

func Transform(model Model) (RestModel, error) {
	var rem *equipable.RestModel
	if model.Equipable != nil {
		m, err := equipable.Transform(*model.Equipable)
		if err != nil {
			return RestModel{}, err
		}
		rem = &m
	}

	rm := RestModel{
		Position:  model.Position,
		Equipable: rem,
	}
	return rm, nil
}

func Extract(model RestModel) (Model, error) {
	m := Model{Position: model.Position}
	if model.Equipable != nil {
		e, err := equipable.Extract(*model.Equipable)
		if err != nil {
			return m, err
		}
		m.Equipable = &e
	}
	return m, nil
}
