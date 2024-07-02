package equipment

import (
	"atlas-character/equipment/slot"
)

type RestModel struct {
	Hat      slot.RestModel `json:"hat"`
	Medal    slot.RestModel `json:"medal"`
	Forehead slot.RestModel `json:"forehead"`
	Ring1    slot.RestModel `json:"ring1"`
	Ring2    slot.RestModel `json:"ring2"`
	Eye      slot.RestModel `json:"eye"`
	Earring  slot.RestModel `json:"earring"`
	Shoulder slot.RestModel `json:"shoulder"`
	Cape     slot.RestModel `json:"cape"`
	Top      slot.RestModel `json:"top"`
	Pendant  slot.RestModel `json:"pendant"`
	Weapon   slot.RestModel `json:"weapon"`
	Shield   slot.RestModel `json:"shield"`
	Gloves   slot.RestModel `json:"gloves"`
	Bottom   slot.RestModel `json:"bottom"`
	Belt     slot.RestModel `json:"belt"`
	Ring3    slot.RestModel `json:"ring3"`
	Ring4    slot.RestModel `json:"ring4"`
	Shoes    slot.RestModel `json:"shoes"`
}

func Transform(m Model) (RestModel, error) {
	hat, err := slot.Transform(m.hat)
	if err != nil {
		return RestModel{}, err
	}
	medal, err := slot.Transform(m.medal)
	if err != nil {
		return RestModel{}, err
	}
	forehead, err := slot.Transform(m.forehead)
	if err != nil {
		return RestModel{}, err
	}
	ring1, err := slot.Transform(m.ring1)
	if err != nil {
		return RestModel{}, err
	}
	ring2, err := slot.Transform(m.ring2)
	if err != nil {
		return RestModel{}, err
	}
	eye, err := slot.Transform(m.eye)
	if err != nil {
		return RestModel{}, err
	}
	earring, err := slot.Transform(m.earring)
	if err != nil {
		return RestModel{}, err
	}
	shoulder, err := slot.Transform(m.shoulder)
	if err != nil {
		return RestModel{}, err
	}
	cape, err := slot.Transform(m.cape)
	if err != nil {
		return RestModel{}, err
	}
	top, err := slot.Transform(m.top)
	if err != nil {
		return RestModel{}, err
	}
	pendant, err := slot.Transform(m.pendant)
	if err != nil {
		return RestModel{}, err
	}
	weapon, err := slot.Transform(m.weapon)
	if err != nil {
		return RestModel{}, err
	}
	shield, err := slot.Transform(m.shield)
	if err != nil {
		return RestModel{}, err
	}
	gloves, err := slot.Transform(m.gloves)
	if err != nil {
		return RestModel{}, err
	}
	bottom, err := slot.Transform(m.bottom)
	if err != nil {
		return RestModel{}, err
	}
	belt, err := slot.Transform(m.belt)
	if err != nil {
		return RestModel{}, err
	}
	ring3, err := slot.Transform(m.ring3)
	if err != nil {
		return RestModel{}, err
	}
	ring4, err := slot.Transform(m.ring4)
	if err != nil {
		return RestModel{}, err
	}
	shoes, err := slot.Transform(m.shoes)
	if err != nil {
		return RestModel{}, err
	}
	rm := RestModel{
		Hat:      hat,
		Medal:    medal,
		Forehead: forehead,
		Ring1:    ring1,
		Ring2:    ring2,
		Eye:      eye,
		Earring:  earring,
		Shoulder: shoulder,
		Cape:     cape,
		Top:      top,
		Pendant:  pendant,
		Weapon:   weapon,
		Shield:   shield,
		Gloves:   gloves,
		Bottom:   bottom,
		Belt:     belt,
		Ring3:    ring3,
		Ring4:    ring4,
		Shoes:    shoes,
	}
	return rm, nil
}

func Extract(m RestModel) (Model, error) {
	hat, err := slot.Extract(m.Hat)
	if err != nil {
		return Model{}, err
	}
	medal, err := slot.Extract(m.Medal)
	if err != nil {
		return Model{}, err
	}
	forehead, err := slot.Extract(m.Forehead)
	if err != nil {
		return Model{}, err
	}
	ring1, err := slot.Extract(m.Ring1)
	if err != nil {
		return Model{}, err
	}
	ring2, err := slot.Extract(m.Ring2)
	if err != nil {
		return Model{}, err
	}
	eye, err := slot.Extract(m.Eye)
	if err != nil {
		return Model{}, err
	}
	earring, err := slot.Extract(m.Earring)
	if err != nil {
		return Model{}, err
	}
	shoulder, err := slot.Extract(m.Shoulder)
	if err != nil {
		return Model{}, err
	}
	cape, err := slot.Extract(m.Cape)
	if err != nil {
		return Model{}, err
	}
	top, err := slot.Extract(m.Top)
	if err != nil {
		return Model{}, err
	}
	pendant, err := slot.Extract(m.Pendant)
	if err != nil {
		return Model{}, err
	}
	weapon, err := slot.Extract(m.Weapon)
	if err != nil {
		return Model{}, err
	}
	shield, err := slot.Extract(m.Shield)
	if err != nil {
		return Model{}, err
	}
	gloves, err := slot.Extract(m.Gloves)
	if err != nil {
		return Model{}, err
	}
	bottom, err := slot.Extract(m.Bottom)
	if err != nil {
		return Model{}, err
	}
	belt, err := slot.Extract(m.Belt)
	if err != nil {
		return Model{}, err
	}
	ring3, err := slot.Extract(m.Ring3)
	if err != nil {
		return Model{}, err
	}
	ring4, err := slot.Extract(m.Ring4)
	if err != nil {
		return Model{}, err
	}
	shoes, err := slot.Extract(m.Shoes)
	if err != nil {
		return Model{}, err
	}
	return Model{
		hat:      hat,
		medal:    medal,
		forehead: forehead,
		ring1:    ring1,
		ring2:    ring2,
		eye:      eye,
		earring:  earring,
		shoulder: shoulder,
		cape:     cape,
		top:      top,
		pendant:  pendant,
		weapon:   weapon,
		shield:   shield,
		gloves:   gloves,
		bottom:   bottom,
		belt:     belt,
		ring3:    ring3,
		ring4:    ring4,
		shoes:    shoes,
	}, nil
}
