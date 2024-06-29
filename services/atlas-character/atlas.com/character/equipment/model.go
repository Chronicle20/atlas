package equipment

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
)

type Model struct {
	hat      slot.Model
	medal    slot.Model
	forehead slot.Model
	ring1    slot.Model
	ring2    slot.Model
	eye      slot.Model
	earring  slot.Model
	shoulder slot.Model
	cape     slot.Model
	top      slot.Model
	pendant  slot.Model
	weapon   slot.Model
	shield   slot.Model
	gloves   slot.Model
	bottom   slot.Model
	belt     slot.Model
	ring3    slot.Model
	ring4    slot.Model
	shoes    slot.Model
}

func NewModel() Model {
	m := Model{
		hat:      slot.Model{Position: slot.PositionHat},
		medal:    slot.Model{Position: slot.PositionMedal},
		forehead: slot.Model{Position: slot.PositionForehead},
		ring1:    slot.Model{Position: slot.PositionRing1},
		ring2:    slot.Model{Position: slot.PositionRing2},
		eye:      slot.Model{Position: slot.PositionEye},
		earring:  slot.Model{Position: slot.PositionEarring},
		shoulder: slot.Model{Position: slot.PositionShoulder},
		cape:     slot.Model{Position: slot.PositionCape},
		top:      slot.Model{Position: slot.PositionTop},
		pendant:  slot.Model{Position: slot.PositionPendant},
		weapon:   slot.Model{Position: slot.PositionWeapon},
		shield:   slot.Model{Position: slot.PositionShield},
		gloves:   slot.Model{Position: slot.PositionGloves},
		bottom:   slot.Model{Position: slot.PositionBottom},
		belt:     slot.Model{Position: slot.PositionBelt},
		ring3:    slot.Model{Position: slot.PositionRing3},
		ring4:    slot.Model{Position: slot.PositionRing4},
		shoes:    slot.Model{Position: slot.PositionShoes},
	}
	return m
}

func (m Model) SetHat(e *equipable.Model) Model {
	m.hat = slot.Model{Position: slot.PositionHat, Equipable: e}
	return m
}

func (m Model) SetMedal(e *equipable.Model) Model {
	m.medal = slot.Model{Position: slot.PositionMedal, Equipable: e}
	return m
}

func (m Model) SetForehead(e *equipable.Model) Model {
	m.forehead = slot.Model{Position: slot.PositionForehead, Equipable: e}
	return m
}

func (m Model) SetRing1(e *equipable.Model) Model {
	m.ring1 = slot.Model{Position: slot.PositionRing1, Equipable: e}
	return m
}

func (m Model) SetRing2(e *equipable.Model) Model {
	m.ring2 = slot.Model{Position: slot.PositionRing2, Equipable: e}
	return m
}

func (m Model) SetEye(e *equipable.Model) Model {
	m.eye = slot.Model{Position: slot.PositionEye, Equipable: e}
	return m
}

func (m Model) SetEarring(e *equipable.Model) Model {
	m.earring = slot.Model{Position: slot.PositionEarring, Equipable: e}
	return m
}

func (m Model) SetShoulder(e *equipable.Model) Model {
	m.shoulder = slot.Model{Position: slot.PositionShoulder, Equipable: e}
	return m
}

func (m Model) SetCape(e *equipable.Model) Model {
	m.cape = slot.Model{Position: slot.PositionCape, Equipable: e}
	return m
}

func (m Model) SetTop(e *equipable.Model) Model {
	m.top = slot.Model{Position: slot.PositionTop, Equipable: e}
	return m
}

func (m Model) SetPendant(e *equipable.Model) Model {
	m.pendant = slot.Model{Position: slot.PositionPendant, Equipable: e}
	return m
}

func (m Model) SetWeapon(e *equipable.Model) Model {
	m.weapon = slot.Model{Position: slot.PositionWeapon, Equipable: e}
	return m
}

func (m Model) SetShield(e *equipable.Model) Model {
	m.shield = slot.Model{Position: slot.PositionShield, Equipable: e}
	return m
}

func (m Model) SetGloves(e *equipable.Model) Model {
	m.gloves = slot.Model{Position: slot.PositionGloves, Equipable: e}
	return m
}

func (m Model) SetBottom(e *equipable.Model) Model {
	m.bottom = slot.Model{Position: slot.PositionBottom, Equipable: e}
	return m
}

func (m Model) SetBelt(e *equipable.Model) Model {
	m.belt = slot.Model{Position: slot.PositionBelt, Equipable: e}
	return m
}

func (m Model) SetRing3(e *equipable.Model) Model {
	m.ring3 = slot.Model{Position: slot.PositionRing3, Equipable: e}
	return m
}

func (m Model) SetRing4(e *equipable.Model) Model {
	m.ring4 = slot.Model{Position: slot.PositionRing4, Equipable: e}
	return m
}

func (m Model) SetShoes(e *equipable.Model) Model {
	m.shoes = slot.Model{Position: slot.PositionShoes, Equipable: e}
	return m
}

func (m Model) Apply(equips []equipable.Model) Model {
	var rm = m
	for i := range equips {
		e := &equips[i]
		switch slot.Position(e.Slot()) {
		case slot.PositionHat:
			rm = rm.SetHat(e)
		case slot.PositionMedal:
			rm = rm.SetMedal(e)
		case slot.PositionForehead:
			rm = rm.SetForehead(e)
		case slot.PositionRing1:
			rm = rm.SetRing1(e)
		case slot.PositionRing2:
			rm = rm.SetRing2(e)
		case slot.PositionEye:
			rm = rm.SetEye(e)
		case slot.PositionEarring:
			rm = rm.SetEarring(e)
		case slot.PositionShoulder:
			rm = rm.SetShoulder(e)
		case slot.PositionCape:
			rm = rm.SetCape(e)
		case slot.PositionTop:
			rm = rm.SetTop(e)
		case slot.PositionPendant:
			rm = rm.SetPendant(e)
		case slot.PositionWeapon:
			rm = rm.SetWeapon(e)
		case slot.PositionShield:
			rm = rm.SetShield(e)
		case slot.PositionGloves:
			rm = rm.SetGloves(e)
		case slot.PositionBottom:
			rm = rm.SetBottom(e)
		case slot.PositionBelt:
			rm = rm.SetBelt(e)
		case slot.PositionRing3:
			rm = rm.SetRing3(e)
		case slot.PositionRing4:
			rm = rm.SetRing4(e)
		case slot.PositionShoes:
			rm = rm.SetShoes(e)
		}
	}
	return rm
}
