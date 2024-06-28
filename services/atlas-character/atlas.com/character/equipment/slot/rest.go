package slot

import (
	"atlas-character/equipable"
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
