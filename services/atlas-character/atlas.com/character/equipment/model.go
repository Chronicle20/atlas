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

type SlotSetter func(model *equipable.Model) Model

func (m Model) SetHat(e *equipable.Model) Model {
	m.hat = m.hat.SetEquipable(e)
	return m
}

func (m Model) SetCashHat(e *equipable.Model) Model {
	m.hat = m.hat.SetCashEquipable(e)
	return m
}

func (m Model) SetMedal(e *equipable.Model) Model {
	m.medal = m.medal.SetEquipable(e)
	return m
}

func (m Model) SetCashMedal(e *equipable.Model) Model {
	m.medal = m.medal.SetCashEquipable(e)
	return m
}

func (m Model) SetForehead(e *equipable.Model) Model {
	m.forehead = m.forehead.SetEquipable(e)
	return m
}

func (m Model) SetCashForehead(e *equipable.Model) Model {
	m.forehead = m.forehead.SetCashEquipable(e)
	return m
}

func (m Model) SetRing1(e *equipable.Model) Model {
	m.ring1 = m.ring1.SetEquipable(e)
	return m
}

func (m Model) SetCashRing1(e *equipable.Model) Model {
	m.ring1 = m.ring1.SetCashEquipable(e)
	return m
}

func (m Model) SetRing2(e *equipable.Model) Model {
	m.ring2 = m.ring2.SetEquipable(e)
	return m
}

func (m Model) SetCashRing2(e *equipable.Model) Model {
	m.ring2 = m.ring2.SetCashEquipable(e)
	return m
}

func (m Model) SetEye(e *equipable.Model) Model {
	m.eye = m.eye.SetEquipable(e)
	return m
}

func (m Model) SetCashEye(e *equipable.Model) Model {
	m.eye = m.eye.SetCashEquipable(e)
	return m
}

func (m Model) SetEarring(e *equipable.Model) Model {
	m.earring = m.earring.SetEquipable(e)
	return m
}

func (m Model) SetCashEarring(e *equipable.Model) Model {
	m.earring = m.earring.SetCashEquipable(e)
	return m
}

func (m Model) SetShoulder(e *equipable.Model) Model {
	m.shoulder = m.shoulder.SetEquipable(e)
	return m
}

func (m Model) SetCashShoulder(e *equipable.Model) Model {
	m.shoulder = m.shoulder.SetCashEquipable(e)
	return m
}

func (m Model) SetCape(e *equipable.Model) Model {
	m.cape = m.cape.SetEquipable(e)
	return m
}

func (m Model) SetCashCape(e *equipable.Model) Model {
	m.cape = m.cape.SetCashEquipable(e)
	return m
}

func (m Model) SetTop(e *equipable.Model) Model {
	m.top = m.top.SetEquipable(e)
	return m
}

func (m Model) SetCashTop(e *equipable.Model) Model {
	m.top = m.top.SetCashEquipable(e)
	return m
}

func (m Model) SetPendant(e *equipable.Model) Model {
	m.pendant = m.pendant.SetEquipable(e)
	return m
}

func (m Model) SetCashPendant(e *equipable.Model) Model {
	m.pendant = m.pendant.SetCashEquipable(e)
	return m
}

func (m Model) SetWeapon(e *equipable.Model) Model {
	m.weapon = m.weapon.SetEquipable(e)
	return m
}

func (m Model) SetCashWeapon(e *equipable.Model) Model {
	m.weapon = m.weapon.SetCashEquipable(e)
	return m
}

func (m Model) SetShield(e *equipable.Model) Model {
	m.shield = m.shield.SetEquipable(e)
	return m
}

func (m Model) SetCashShield(e *equipable.Model) Model {
	m.shield = m.shield.SetCashEquipable(e)
	return m
}

func (m Model) SetGloves(e *equipable.Model) Model {
	m.gloves = m.gloves.SetEquipable(e)
	return m
}

func (m Model) SetCashGloves(e *equipable.Model) Model {
	m.gloves = m.gloves.SetCashEquipable(e)
	return m
}

func (m Model) SetBottom(e *equipable.Model) Model {
	m.bottom = m.bottom.SetEquipable(e)
	return m
}

func (m Model) SetCashBottom(e *equipable.Model) Model {
	m.bottom = m.bottom.SetCashEquipable(e)
	return m
}

func (m Model) SetBelt(e *equipable.Model) Model {
	m.belt = m.belt.SetEquipable(e)
	return m
}

func (m Model) SetCashBelt(e *equipable.Model) Model {
	m.belt = m.belt.SetCashEquipable(e)
	return m
}

func (m Model) SetRing3(e *equipable.Model) Model {
	m.ring3 = m.ring3.SetEquipable(e)
	return m
}

func (m Model) SetCashRing3(e *equipable.Model) Model {
	m.ring3 = m.ring3.SetCashEquipable(e)
	return m
}

func (m Model) SetRing4(e *equipable.Model) Model {
	m.ring4 = m.ring4.SetEquipable(e)
	return m
}

func (m Model) SetCashRing4(e *equipable.Model) Model {
	m.ring4 = m.ring4.SetCashEquipable(e)
	return m
}

func (m Model) SetShoes(e *equipable.Model) Model {
	m.shoes = m.shoes.SetEquipable(e)
	return m
}

func (m Model) SetCashShoes(e *equipable.Model) Model {
	m.shoes = m.shoes.SetCashEquipable(e)
	return m
}
