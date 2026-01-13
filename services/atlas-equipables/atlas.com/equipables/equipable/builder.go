package equipable

import "time"

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
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

func NewBuilder(id uint32) *ModelBuilder {
	return &ModelBuilder{id: id}
}

func (b *ModelBuilder) SetItemId(itemId uint32) *ModelBuilder {
	b.itemId = itemId
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

func (b *ModelBuilder) SetHp(hp uint16) *ModelBuilder {
	b.hp = hp
	return b
}

func (b *ModelBuilder) SetMp(mp uint16) *ModelBuilder {
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

func (b *ModelBuilder) AddStrength(delta int16) *ModelBuilder {
	b.strength = addUint16(b.strength, int(delta))
	return b
}

func (b *ModelBuilder) AddDexterity(delta int16) *ModelBuilder {
	b.dexterity = addUint16(b.dexterity, int(delta))
	return b
}

func (b *ModelBuilder) AddIntelligence(delta int16) *ModelBuilder {
	b.intelligence = addUint16(b.intelligence, int(delta))
	return b
}

func (b *ModelBuilder) AddLuck(delta int16) *ModelBuilder {
	b.luck = addUint16(b.luck, int(delta))
	return b
}

func (b *ModelBuilder) AddHp(delta int16) *ModelBuilder {
	b.hp = addUint16(b.hp, int(delta))
	return b
}

func (b *ModelBuilder) AddMp(delta int16) *ModelBuilder {
	b.mp = addUint16(b.mp, int(delta))
	return b
}

func (b *ModelBuilder) AddWeaponAttack(delta int16) *ModelBuilder {
	b.weaponAttack = addUint16(b.weaponAttack, int(delta))
	return b
}

func (b *ModelBuilder) AddMagicAttack(delta int16) *ModelBuilder {
	b.magicAttack = addUint16(b.magicAttack, int(delta))
	return b
}

func (b *ModelBuilder) AddWeaponDefense(delta int16) *ModelBuilder {
	b.weaponDefense = addUint16(b.weaponDefense, int(delta))
	return b
}

func (b *ModelBuilder) AddMagicDefense(delta int16) *ModelBuilder {
	b.magicDefense = addUint16(b.magicDefense, int(delta))
	return b
}

func (b *ModelBuilder) AddAccuracy(delta int16) *ModelBuilder {
	b.accuracy = addUint16(b.accuracy, int(delta))
	return b
}

func (b *ModelBuilder) AddAvoidability(delta int16) *ModelBuilder {
	b.avoidability = addUint16(b.avoidability, int(delta))
	return b
}

func (b *ModelBuilder) AddHands(delta int16) *ModelBuilder {
	b.hands = addUint16(b.hands, int(delta))
	return b
}

func (b *ModelBuilder) AddSpeed(delta int16) *ModelBuilder {
	b.speed = addUint16(b.speed, int(delta))
	return b
}

func (b *ModelBuilder) AddJump(delta int16) *ModelBuilder {
	b.jump = addUint16(b.jump, int(delta))
	return b
}

func (b *ModelBuilder) AddSlots(delta int16) *ModelBuilder {
	b.slots = addUint16(b.slots, int(delta))
	return b
}

func (b *ModelBuilder) AddLevel(delta int8) *ModelBuilder {
	b.level = addByte(b.level, int(delta))
	return b
}

func (b *ModelBuilder) AddExperience(delta int32) *ModelBuilder {
	b.experience = addUint32(b.experience, int(delta))
	return b
}

func (b *ModelBuilder) AddHammersApplied(delta int32) *ModelBuilder {
	b.hammersApplied = addUint32(b.hammersApplied, int(delta))
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:             b.id,
		itemId:         b.itemId,
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

func addUint16(orig uint16, delta int) uint16 {
	v := int(orig) + delta
	if v < 0 {
		return 0
	}
	if v > int(^uint16(0)) {
		return ^uint16(0)
	}
	return uint16(v)
}

func addUint32(orig uint32, delta int) uint32 {
	v := int64(orig) + int64(delta)
	if v < 0 {
		return 0
	}
	if v > int64(^uint32(0)) {
		return ^uint32(0)
	}
	return uint32(v)
}

func addByte(orig byte, delta int) byte {
	v := int(orig) + delta
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}
