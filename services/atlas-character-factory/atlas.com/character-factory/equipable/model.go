package equipable

type Model struct {
	id            uint32
	itemId        uint32
	slot          int16
	referenceId   uint32
	strength      uint16
	dexterity     uint16
	intelligence  uint16
	luck          uint16
	hp            uint16
	mp            uint16
	weaponAttack  uint16
	magicAttack   uint16
	weaponDefense uint16
	magicDefense  uint16
	accuracy      uint16
	avoidability  uint16
	hands         uint16
	speed         uint16
	jump          uint16
	slots         uint16
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) ReferenceId() uint32 {
	return m.referenceId
}
