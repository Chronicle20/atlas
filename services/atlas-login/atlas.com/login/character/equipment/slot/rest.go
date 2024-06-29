package slot

import (
	"atlas-login/character/inventory/equipable"
)

type RestModel struct {
	Position  Position             `json:"position"`
	Equipable *equipable.RestModel `json:"equipable"`
}

func Transform(model Model) RestModel {
	var rem *equipable.RestModel
	if model.Equipable != nil {
		m := equipable.Transform(*model.Equipable)
		rem = &m
	}

	rm := RestModel{
		Position:  model.Position,
		Equipable: rem,
	}
	return rm
}

func Extract(model RestModel) Model {
	if model.Equipable != nil {
		e := equipable.Extract(*model.Equipable)
		return Model{
			Position:  model.Position,
			Equipable: &e,
		}
	} else {
		return Model{Position: model.Position}
	}
}
