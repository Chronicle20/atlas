package equipable

import "time"

type Model struct {
	id             uint32
	slot           int16
	itemId         uint32
	strength       uint16
	dexterity      uint16
	intelligence   uint16
	luck           uint16
	hp             uint16
	mp             uint16
	weaponAttack   uint16
	magicAttack    uint16
	weaponDefense  uint16
	magicDefense   uint16
	accuracy       uint16
	avoidability   uint16
	hands          uint16
	speed          uint16
	jump           uint16
	slots          uint16
	ownerName      string
	locked         bool
	spikes         bool
	karmaUsed      bool
	cold           bool
	canBeTraded    bool
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	expiration     time.Time
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

func (m Model) Slots() uint16 {
	return m.slots
}

func (m Model) OwnerName() string {
	return m.ownerName
}

func (m Model) Locked() bool {
	return m.locked
}

func (m Model) Spikes() bool {
	return m.spikes
}

func (m Model) KarmaUsed() bool {
	return m.karmaUsed
}

func (m Model) Cold() bool {
	return m.cold
}

func (m Model) CanBeTraded() bool {
	return m.canBeTraded
}

func (m Model) LevelType() byte {
	return m.levelType
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) Experience() uint32 {
	return m.experience
}

func (m Model) HammersApplied() uint32 {
	return m.hammersApplied
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) Flags() uint16 {
	flag := uint16(0)
	if m.Locked() {
		flag |= 0x01
	}
	if m.Spikes() {
		flag |= 0x02
	}
	if m.Cold() {
		flag |= 0x04
	}
	if m.CanBeTraded() {
		flag |= 0x08
	}
	if m.KarmaUsed() {
		flag |= 0x10
	}
	return flag
}

type ModelBuilder struct {
	id             uint32
	itemId         uint32
	slot           int16
	strength       uint16
	dexterity      uint16
	intelligence   uint16
	luck           uint16
	hp             uint16
	mp             uint16
	weaponAttack   uint16
	magicAttack    uint16
	weaponDefense  uint16
	magicDefense   uint16
	accuracy       uint16
	avoidability   uint16
	hands          uint16
	speed          uint16
	jump           uint16
	slots          uint16
	ownerName      string
	locked         bool
	spikes         bool
	karmaUsed      bool
	cold           bool
	canBeTraded    bool
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	expiration     time.Time
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

func CloneFromModel(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:             m.Id(),
		itemId:         m.ItemId(),
		slot:           m.Slot(),
		strength:       m.Strength(),
		dexterity:      m.Dexterity(),
		intelligence:   m.Intelligence(),
		luck:           m.Luck(),
		hp:             m.HP(),
		mp:             m.MP(),
		weaponAttack:   m.WeaponAttack(),
		magicAttack:    m.MagicAttack(),
		weaponDefense:  m.WeaponDefense(),
		magicDefense:   m.MagicDefense(),
		accuracy:       m.Accuracy(),
		avoidability:   m.Avoidability(),
		hands:          m.Hands(),
		speed:          m.Speed(),
		jump:           m.Jump(),
		slots:          m.Slots(),
		ownerName:      m.OwnerName(),
		locked:         m.Locked(),
		spikes:         m.Spikes(),
		karmaUsed:      m.KarmaUsed(),
		cold:           m.Cold(),
		canBeTraded:    m.CanBeTraded(),
		levelType:      m.LevelType(),
		level:          m.Level(),
		experience:     m.Experience(),
		hammersApplied: m.HammersApplied(),
		expiration:     m.Expiration(),
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

func (b *ModelBuilder) SetOwnerName(name string) *ModelBuilder {
	b.ownerName = name
	return b
}

func (b *ModelBuilder) SetLocked(locked bool) *ModelBuilder {
	b.locked = locked
	return b
}

func (b *ModelBuilder) SetSpikes(spikes bool) *ModelBuilder {
	b.spikes = spikes
	return b
}

func (b *ModelBuilder) SetKarmaUsed(karmaUsed bool) *ModelBuilder {
	b.karmaUsed = karmaUsed
	return b
}

func (b *ModelBuilder) SetCold(cold bool) *ModelBuilder {
	b.cold = cold
	return b
}

func (b *ModelBuilder) SetCanBeTraded(canBeTraded bool) *ModelBuilder {
	b.canBeTraded = canBeTraded
	return b
}

func (b *ModelBuilder) SetLevelType(levelType byte) *ModelBuilder {
	b.levelType = levelType
	return b
}

func (b *ModelBuilder) SetLevel(level byte) *ModelBuilder {
	b.level = level
	return b
}

func (b *ModelBuilder) SetExperience(exp uint32) *ModelBuilder {
	b.experience = exp
	return b
}

func (b *ModelBuilder) SetHammersApplied(hammers uint32) *ModelBuilder {
	b.hammersApplied = hammers
	return b
}

func (b *ModelBuilder) SetExpiration(expiration time.Time) *ModelBuilder {
	b.expiration = expiration
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:             b.id,
		itemId:         b.itemId,
		slot:           b.slot,
		strength:       b.strength,
		dexterity:      b.dexterity,
		intelligence:   b.intelligence,
		luck:           b.luck,
		hp:             b.hp,
		mp:             b.mp,
		weaponAttack:   b.weaponAttack,
		magicAttack:    b.magicAttack,
		weaponDefense:  b.weaponDefense,
		magicDefense:   b.magicDefense,
		accuracy:       b.accuracy,
		avoidability:   b.avoidability,
		hands:          b.hands,
		speed:          b.speed,
		jump:           b.jump,
		slots:          b.slots,
		ownerName:      b.ownerName,
		locked:         b.locked,
		spikes:         b.spikes,
		karmaUsed:      b.karmaUsed,
		cold:           b.cold,
		canBeTraded:    b.canBeTraded,
		levelType:      b.levelType,
		level:          b.level,
		experience:     b.experience,
		hammersApplied: b.hammersApplied,
		expiration:     b.expiration,
	}
}
