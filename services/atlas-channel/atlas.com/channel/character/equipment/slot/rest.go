package slot

import (
	"atlas-channel/character/inventory/equipable"
)

type RestModel struct {
	Position      Position             `json:"position"`
	Equipable     *equipable.RestModel `json:"equipable"`
	CashEquipable *equipable.RestModel `json:"cashEquipable"`
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

func Extract(model RestModel) (Model, error) {
	m := Model{Position: model.Position}
	if model.Equipable != nil {
		e, err := equipable.Extract(*model.Equipable)
		if err != nil {
			return m, err
		}
		m.Equipable = &e
	}
	if model.CashEquipable != nil {
		e, err := equipable.Extract(*model.CashEquipable)
		if err != nil {
			return m, err
		}
		m.CashEquipable = &e
	}
	return m, nil
}
