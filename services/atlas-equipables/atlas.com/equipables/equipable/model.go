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

func (m Model) Id() uint32 {
	return m.id
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
