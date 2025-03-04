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

func (m Model) Quantity() uint32 {
	return 1
}

func (m Model) ReferenceId() uint32 {
	return m.referenceId
}

func (m Model) Strength() uint16 {
	return m.strength
}

func ReferenceId(m Model) (uint32, error) {
	return m.ReferenceId(), nil
}

type ModelBuilder struct {
	model Model
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

func (b *ModelBuilder) SetID(id uint32) *ModelBuilder {
	b.model.id = id
	return b
}

func (b *ModelBuilder) SetItemId(itemId uint32) *ModelBuilder {
	b.model.itemId = itemId
	return b
}

func (b *ModelBuilder) SetSlot(slot int16) *ModelBuilder {
	b.model.slot = slot
	return b
}

func (b *ModelBuilder) SetReferenceId(referenceId uint32) *ModelBuilder {
	b.model.referenceId = referenceId
	return b
}

func (b *ModelBuilder) SetStrength(strength uint16) *ModelBuilder {
	b.model.strength = strength
	return b
}

func (b *ModelBuilder) SetDexterity(dexterity uint16) *ModelBuilder {
	b.model.dexterity = dexterity
	return b
}

func (b *ModelBuilder) SetIntelligence(intelligence uint16) *ModelBuilder {
	b.model.intelligence = intelligence
	return b
}

func (b *ModelBuilder) SetLuck(luck uint16) *ModelBuilder {
	b.model.luck = luck
	return b
}

func (b *ModelBuilder) SetHP(hp uint16) *ModelBuilder {
	b.model.hp = hp
	return b
}

func (b *ModelBuilder) SetMP(mp uint16) *ModelBuilder {
	b.model.mp = mp
	return b
}

func (b *ModelBuilder) SetWeaponAttack(weaponAttack uint16) *ModelBuilder {
	b.model.weaponAttack = weaponAttack
	return b
}

func (b *ModelBuilder) SetMagicAttack(magicAttack uint16) *ModelBuilder {
	b.model.magicAttack = magicAttack
	return b
}

func (b *ModelBuilder) SetWeaponDefense(weaponDefense uint16) *ModelBuilder {
	b.model.weaponDefense = weaponDefense
	return b
}

func (b *ModelBuilder) SetMagicDefense(magicDefense uint16) *ModelBuilder {
	b.model.magicDefense = magicDefense
	return b
}

func (b *ModelBuilder) SetAccuracy(accuracy uint16) *ModelBuilder {
	b.model.accuracy = accuracy
	return b
}

func (b *ModelBuilder) SetAvoidability(avoidability uint16) *ModelBuilder {
	b.model.avoidability = avoidability
	return b
}

func (b *ModelBuilder) SetHands(hands uint16) *ModelBuilder {
	b.model.hands = hands
	return b
}

func (b *ModelBuilder) SetSpeed(speed uint16) *ModelBuilder {
	b.model.speed = speed
	return b
}

func (b *ModelBuilder) SetJump(jump uint16) *ModelBuilder {
	b.model.jump = jump
	return b
}

func (b *ModelBuilder) SetSlots(slots uint16) *ModelBuilder {
	b.model.slots = slots
	return b
}

func (b *ModelBuilder) Build() Model {
	return b.model
}
