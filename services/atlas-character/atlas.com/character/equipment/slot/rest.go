package slot

import (
	"atlas-character/equipable"
)

type RestModel struct {
	Position      Position             `json:"position"`
	Equipable     *equipable.RestModel `json:"equipable"`
	CashEquipable *equipable.RestModel `json:"cashEquipable"`
}

func Transform(model Model) (RestModel, error) {
	var rem *equipable.RestModel
	var rcem *equipable.RestModel
	if model.Equipable != nil {
		m, err := equipable.Transform(*model.Equipable)
		if err != nil {
			return RestModel{}, err
		}
		rem = &m
	}
	if model.CashEquipable != nil {
		m, err := equipable.Transform(*model.CashEquipable)
		if err != nil {
			return RestModel{}, err
		}
		rcem = &m
	}

	rm := RestModel{
		Position:      model.Position,
		Equipable:     rem,
		CashEquipable: rcem,
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
	if model.CashEquipable != nil {
		e, err := equipable.Extract(*model.CashEquipable)
		if err != nil {
			return m, err
		}
		m.CashEquipable = &e
	}
	return m, nil
}
