package equipment

import "atlas-login/character/equipment/slot"

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

func (m Model) Hat() slot.Model {
	return m.hat
}

func (m Model) Medal() slot.Model {
	return m.medal
}

func (m Model) Forehead() slot.Model {
	return m.forehead
}

func (m Model) Ring1() slot.Model {
	return m.ring1
}

func (m Model) Ring2() slot.Model {
	return m.ring2
}

func (m Model) Eye() slot.Model {
	return m.eye
}

func (m Model) Earring() slot.Model {
	return m.earring
}

func (m Model) Shoulder() slot.Model {
	return m.shoulder
}

func (m Model) Cape() slot.Model {
	return m.cape
}

func (m Model) Top() slot.Model {
	return m.top
}

func (m Model) Pendant() slot.Model {
	return m.pendant
}

func (m Model) Weapon() slot.Model {
	return m.weapon
}

func (m Model) Shield() slot.Model {
	return m.shield
}

func (m Model) Gloves() slot.Model {
	return m.gloves
}

func (m Model) Bottom() slot.Model {
	return m.bottom
}

func (m Model) Belt() slot.Model {
	return m.belt
}

func (m Model) Ring3() slot.Model {
	return m.ring3
}

func (m Model) Ring4() slot.Model {
	return m.ring4
}

func (m Model) Shoes() slot.Model {
	return m.shoes
}
