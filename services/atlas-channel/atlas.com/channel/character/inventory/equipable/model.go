package equipable

type Model struct {
	id            uint32
	slot          int16
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
	hammersUsed   uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Expiration() int64 {
	return -1
}

func (m Model) Slots() uint16 {
	return m.slots
}

func (m Model) Strength() uint16 {
	return m.strength
}

func (m Model) Dexterity() uint16 {
	return m.dexterity
}

func (m Model) Intelligence() uint16 {
	return m.intelligence
}

func (m Model) Luck() uint16 {
	return m.luck
}

func (m Model) HP() uint16 {
	return m.hp
}

func (m Model) MP() uint16 {
	return m.mp
}

func (m Model) WeaponAttack() uint16 {
	return m.weaponAttack
}

func (m Model) MagicAttack() uint16 {
	return m.magicAttack
}

func (m Model) WeaponDefense() uint16 {
	return m.weaponDefense
}

func (m Model) MagicDefense() uint16 {
	return m.magicDefense
}

func (m Model) Accuracy() uint16 {
	return m.accuracy
}

func (m Model) Avoidability() uint16 {
	return m.avoidability
}

func (m Model) Hands() uint16 {
	return m.hands
}

func (m Model) Speed() uint16 {
	return m.speed
}

func (m Model) Jump() uint16 {
	return m.jump
}

func (m Model) OwnerName() string {
	return ""
}

func (m Model) Flags() uint16 {
	return 0
}

func (m Model) LevelUpType() byte {
	return 0
}

func (m Model) Level() byte {
	return 0
}

func (m Model) Experience() uint32 {
	return 0
}

func (m Model) ViciousHammer() int32 {
	return 0
}

type ModelBuilder struct {
	id            uint32
	itemId        uint32
	slot          int16
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

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

func CloneFromModel(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:            m.Id(),
		itemId:        m.ItemId(),
		slot:          m.Slot(),
		strength:      m.Strength(),
		dexterity:     m.Dexterity(),
		intelligence:  m.Intelligence(),
		luck:          m.Luck(),
		hp:            m.HP(),
		mp:            m.MP(),
		weaponAttack:  m.WeaponAttack(),
		magicAttack:   m.MagicAttack(),
		weaponDefense: m.WeaponDefense(),
		magicDefense:  m.MagicDefense(),
		accuracy:      m.Accuracy(),
		avoidability:  m.Avoidability(),
		hands:         m.Hands(),
		speed:         m.Speed(),
		jump:          m.Jump(),
		slots:         m.Slots(),
	}
}

func (b *ModelBuilder) SetID(id uint32) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) SetItemId(itemId uint32) *ModelBuilder {
	b.itemId = itemId
	return b
}

func (b *ModelBuilder) SetSlot(slot int16) *ModelBuilder {
	b.slot = slot
	return b
}

func (b *ModelBuilder) SetStrength(strength uint16) *ModelBuilder {
	b.strength = strength
	return b
}

func (b *ModelBuilder) SetDexterity(dexterity uint16) *ModelBuilder {
	b.dexterity = dexterity
	return b
}

func (b *ModelBuilder) SetIntelligence(intelligence uint16) *ModelBuilder {
	b.intelligence = intelligence
	return b
}

func (b *ModelBuilder) SetLuck(luck uint16) *ModelBuilder {
	b.luck = luck
	return b
}

func (b *ModelBuilder) SetHP(hp uint16) *ModelBuilder {
	b.hp = hp
	return b
}

func (b *ModelBuilder) SetMP(mp uint16) *ModelBuilder {
	b.mp = mp
	return b
}

func (b *ModelBuilder) SetWeaponAttack(weaponAttack uint16) *ModelBuilder {
	b.weaponAttack = weaponAttack
	return b
}

func (b *ModelBuilder) SetMagicAttack(magicAttack uint16) *ModelBuilder {
	b.magicAttack = magicAttack
	return b
}

func (b *ModelBuilder) SetWeaponDefense(weaponDefense uint16) *ModelBuilder {
	b.weaponDefense = weaponDefense
	return b
}

func (b *ModelBuilder) SetMagicDefense(magicDefense uint16) *ModelBuilder {
	b.magicDefense = magicDefense
	return b
}

func (b *ModelBuilder) SetAccuracy(accuracy uint16) *ModelBuilder {
	b.accuracy = accuracy
	return b
}

func (b *ModelBuilder) SetAvoidability(avoidability uint16) *ModelBuilder {
	b.avoidability = avoidability
	return b
}

func (b *ModelBuilder) SetHands(hands uint16) *ModelBuilder {
	b.hands = hands
	return b
}

func (b *ModelBuilder) SetSpeed(speed uint16) *ModelBuilder {
	b.speed = speed
	return b
}

func (b *ModelBuilder) SetJump(jump uint16) *ModelBuilder {
	b.jump = jump
	return b
}

func (b *ModelBuilder) SetSlots(slots uint16) *ModelBuilder {
	b.slots = slots
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:            b.id,
		itemId:        b.itemId,
		slot:          b.slot,
		strength:      b.strength,
		dexterity:     b.dexterity,
		intelligence:  b.intelligence,
		luck:          b.luck,
		hp:            b.hp,
		mp:            b.mp,
		weaponAttack:  b.weaponAttack,
		magicAttack:   b.magicAttack,
		weaponDefense: b.weaponDefense,
		magicDefense:  b.magicDefense,
		accuracy:      b.accuracy,
		avoidability:  b.avoidability,
		hands:         b.hands,
		speed:         b.speed,
		jump:          b.jump,
		slots:         b.slots,
	}
}
