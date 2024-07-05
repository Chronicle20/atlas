package statistics

type Model struct {
	itemId        uint32
	strength      uint16
	dexterity     uint16
	intelligence  uint16
	luck          uint16
	weaponAttack  uint16
	weaponDefense uint16
	magicAttack   uint16
	magicDefense  uint16
	accuracy      uint16
	avoidability  uint16
	speed         uint16
	jump          uint16
	hp            uint16
	mp            uint16
	slots         uint16
	cash          bool
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:            m.itemId,
		Strength:      m.strength,
		Dexterity:     m.dexterity,
		Intelligence:  m.intelligence,
		Luck:          m.luck,
		HP:            m.hp,
		MP:            m.mp,
		WeaponAttack:  m.weaponAttack,
		MagicAttack:   m.magicAttack,
		WeaponDefense: m.weaponDefense,
		MagicDefense:  m.magicDefense,
		Accuracy:      m.accuracy,
		Avoidability:  m.avoidability,
		Speed:         m.speed,
		Jump:          m.jump,
		Slots:         m.slots,
		Cash:          m.cash,
	}, nil
}
