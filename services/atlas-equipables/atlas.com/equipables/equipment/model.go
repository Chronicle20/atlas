package equipment

type Model struct {
	id            uint32
	itemId        uint32
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

func (e Model) Id() uint32 {
	return e.id
}

func (e Model) ItemId() uint32 {
	return e.itemId
}

func (e Model) Strength() uint16 {
	return e.strength
}

func (e Model) Dexterity() uint16 {
	return e.dexterity
}

func (e Model) Intelligence() uint16 {
	return e.intelligence
}

func (e Model) Luck() uint16 {
	return e.luck
}

func (e Model) HP() uint16 {
	return e.hp
}

func (e Model) MP() uint16 {
	return e.mp
}

func (e Model) WeaponAttack() uint16 {
	return e.weaponAttack
}

func (e Model) MagicAttack() uint16 {
	return e.magicAttack
}

func (e Model) WeaponDefense() uint16 {
	return e.weaponDefense
}

func (e Model) MagicDefense() uint16 {
	return e.magicDefense
}

func (e Model) Accuracy() uint16 {
	return e.accuracy
}

func (e Model) Avoidability() uint16 {
	return e.avoidability
}

func (e Model) Hands() uint16 {
	return e.hands
}

func (e Model) Speed() uint16 {
	return e.speed
}

func (e Model) Jump() uint16 {
	return e.jump
}

func (e Model) Slots() uint16 {
	return e.slots
}

type ModelBuilder struct {
	id            uint32
	itemId        uint32
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

func NewBuilder(id uint32) *ModelBuilder {
	return &ModelBuilder{id: id}
}

func (e *ModelBuilder) SetItemId(itemId uint32) *ModelBuilder {
	e.itemId = itemId
	return e
}

func (e *ModelBuilder) SetStrength(strength uint16) *ModelBuilder {
	e.strength = strength
	return e
}

func (e *ModelBuilder) SetDexterity(dexterity uint16) *ModelBuilder {
	e.dexterity = dexterity
	return e
}

func (e *ModelBuilder) SetIntelligence(intelligence uint16) *ModelBuilder {
	e.intelligence = intelligence
	return e
}

func (e *ModelBuilder) SetLuck(luck uint16) *ModelBuilder {
	e.luck = luck
	return e
}

func (e *ModelBuilder) SetHp(hp uint16) *ModelBuilder {
	e.hp = hp
	return e
}

func (e *ModelBuilder) SetMp(mp uint16) *ModelBuilder {
	e.mp = mp
	return e
}

func (e *ModelBuilder) SetWeaponAttack(weaponAttack uint16) *ModelBuilder {
	e.weaponAttack = weaponAttack
	return e
}

func (e *ModelBuilder) SetMagicAttack(magicAttack uint16) *ModelBuilder {
	e.magicAttack = magicAttack
	return e
}

func (e *ModelBuilder) SetWeaponDefense(weaponDefense uint16) *ModelBuilder {
	e.weaponDefense = weaponDefense
	return e
}

func (e *ModelBuilder) SetMagicDefense(magicDefense uint16) *ModelBuilder {
	e.magicDefense = magicDefense
	return e
}

func (e *ModelBuilder) SetAccuracy(accuracy uint16) *ModelBuilder {
	e.accuracy = accuracy
	return e
}

func (e *ModelBuilder) SetAvoidability(avoidability uint16) *ModelBuilder {
	e.avoidability = avoidability
	return e
}

func (e *ModelBuilder) SetHands(hands uint16) *ModelBuilder {
	e.hands = hands
	return e
}

func (e *ModelBuilder) SetSpeed(speed uint16) *ModelBuilder {
	e.speed = speed
	return e
}

func (e *ModelBuilder) SetJump(jump uint16) *ModelBuilder {
	e.jump = jump
	return e
}

func (e *ModelBuilder) SetSlots(slots uint16) *ModelBuilder {
	e.slots = slots
	return e
}

func (e *ModelBuilder) Build() Model {
	return Model{
		id:            e.id,
		itemId:        e.itemId,
		strength:      e.strength,
		dexterity:     e.dexterity,
		intelligence:  e.intelligence,
		luck:          e.luck,
		hp:            e.hp,
		mp:            e.mp,
		weaponAttack:  e.weaponAttack,
		magicAttack:   e.magicAttack,
		weaponDefense: e.weaponDefense,
		magicDefense:  e.magicDefense,
		accuracy:      e.accuracy,
		avoidability:  e.avoidability,
		hands:         e.hands,
		speed:         e.speed,
		jump:          e.jump,
		slots:         e.slots,
	}
}
