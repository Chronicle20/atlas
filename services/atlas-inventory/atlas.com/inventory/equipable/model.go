package equipable

import "time"

type Model struct {
	id             uint32
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

func (m Model) WeaponAttack() uint16 {
	return m.weaponAttack
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) HP() uint16 {
	return m.hp
}

func (m Model) MP() uint16 {
	return m.mp
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

func (m Model) OwnerId() uint32 {
	// TODO
	return 0
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

type ModelBuilder struct {
	id             uint32
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

func (m *ModelBuilder) SetItemId(itemId uint32) *ModelBuilder {
	m.itemId = itemId
	return m
}

func (m *ModelBuilder) SetStrength(strength uint16) *ModelBuilder {
	m.strength = strength
	return m
}

func (m *ModelBuilder) SetDexterity(dexterity uint16) *ModelBuilder {
	m.dexterity = dexterity
	return m
}

func (m *ModelBuilder) SetIntelligence(intelligence uint16) *ModelBuilder {
	m.intelligence = intelligence
	return m
}

func (m *ModelBuilder) SetLuck(luck uint16) *ModelBuilder {
	m.luck = luck
	return m
}

func (m *ModelBuilder) SetHp(hp uint16) *ModelBuilder {
	m.hp = hp
	return m
}

func (m *ModelBuilder) SetMp(mp uint16) *ModelBuilder {
	m.mp = mp
	return m
}

func (m *ModelBuilder) SetWeaponAttack(weaponAttack uint16) *ModelBuilder {
	m.weaponAttack = weaponAttack
	return m
}

func (m *ModelBuilder) SetMagicAttack(magicAttack uint16) *ModelBuilder {
	m.magicAttack = magicAttack
	return m
}

func (m *ModelBuilder) SetWeaponDefense(weaponDefense uint16) *ModelBuilder {
	m.weaponDefense = weaponDefense
	return m
}

func (m *ModelBuilder) SetMagicDefense(magicDefense uint16) *ModelBuilder {
	m.magicDefense = magicDefense
	return m
}

func (m *ModelBuilder) SetAccuracy(accuracy uint16) *ModelBuilder {
	m.accuracy = accuracy
	return m
}

func (m *ModelBuilder) SetAvoidability(avoidability uint16) *ModelBuilder {
	m.avoidability = avoidability
	return m
}

func (m *ModelBuilder) SetHands(hands uint16) *ModelBuilder {
	m.hands = hands
	return m
}

func (m *ModelBuilder) SetSpeed(speed uint16) *ModelBuilder {
	m.speed = speed
	return m
}

func (m *ModelBuilder) SetJump(jump uint16) *ModelBuilder {
	m.jump = jump
	return m
}

func (m *ModelBuilder) SetSlots(slots uint16) *ModelBuilder {
	m.slots = slots
	return m
}

func (m *ModelBuilder) SetOwnerName(ownerName string) *ModelBuilder {
	m.ownerName = ownerName
	return m
}

func (m *ModelBuilder) SetLocked(locked bool) *ModelBuilder {
	m.locked = locked
	return m
}

func (m *ModelBuilder) SetSpikes(spikes bool) *ModelBuilder {
	m.spikes = spikes
	return m
}

func (m *ModelBuilder) SetKarmaUsed(karmaUsed bool) *ModelBuilder {
	m.karmaUsed = karmaUsed
	return m
}

func (m *ModelBuilder) SetCold(cold bool) *ModelBuilder {
	m.cold = cold
	return m
}

func (m *ModelBuilder) SetCanBeTraded(canBeTraded bool) *ModelBuilder {
	m.canBeTraded = canBeTraded
	return m
}

func (m *ModelBuilder) SetLevelType(levelType byte) *ModelBuilder {
	m.levelType = levelType
	return m
}

func (m *ModelBuilder) SetLevel(level byte) *ModelBuilder {
	m.level = level
	return m
}

func (m *ModelBuilder) SetExperience(experience uint32) *ModelBuilder {
	m.experience = experience
	return m
}

func (m *ModelBuilder) SetHammersApplied(hammersApplied uint32) *ModelBuilder {
	m.hammersApplied = hammersApplied
	return m
}

func (m *ModelBuilder) SetExpiration(expiration time.Time) *ModelBuilder {
	m.expiration = expiration
	return m
}

func (m *ModelBuilder) Build() Model {
	return Model{
		id:             m.id,
		itemId:         m.itemId,
		strength:       m.strength,
		dexterity:      m.dexterity,
		intelligence:   m.intelligence,
		luck:           m.luck,
		hp:             m.hp,
		mp:             m.mp,
		weaponAttack:   m.weaponAttack,
		magicAttack:    m.magicAttack,
		weaponDefense:  m.weaponDefense,
		magicDefense:   m.magicDefense,
		accuracy:       m.accuracy,
		avoidability:   m.avoidability,
		hands:          m.hands,
		speed:          m.speed,
		jump:           m.jump,
		slots:          m.slots,
		ownerName:      m.ownerName,
		locked:         m.locked,
		spikes:         m.spikes,
		karmaUsed:      m.karmaUsed,
		cold:           m.cold,
		canBeTraded:    m.canBeTraded,
		levelType:      m.levelType,
		level:          m.level,
		experience:     m.experience,
		hammersApplied: m.hammersApplied,
		expiration:     m.expiration,
	}
}
