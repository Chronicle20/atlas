package asset

import (
	"time"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

type Builder[E any] struct {
	id            uint32
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType ReferenceType
	referenceData E
}

func NewBuilder[E any](id uint32, templateId uint32, referenceId uint32, referenceType ReferenceType) *Builder[E] {
	return &Builder[E]{
		id:            id,
		slot:          0,
		templateId:    templateId,
		expiration:    time.Time{},
		referenceId:   referenceId,
		referenceType: referenceType,
	}
}

func (b *Builder[E]) SetSlot(slot int16) *Builder[E] {
	b.slot = slot
	return b
}

func (b *Builder[E]) SetExpiration(e time.Time) *Builder[E] {
	b.expiration = e
	return b
}

func (b *Builder[E]) SetReferenceData(e E) *Builder[E] {
	b.referenceData = e
	return b
}

func (b *Builder[E]) Build() Model[E] {
	return Model[E]{
		id:            b.id,
		slot:          b.slot,
		templateId:    b.templateId,
		expiration:    b.expiration,
		referenceId:   b.referenceId,
		referenceType: b.referenceType,
		referenceData: b.referenceData,
	}
}

type EquipableReferenceDataBuilder struct {
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
	ownerId        uint32
	flag           uint16
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	expiration     time.Time
}

// NewEquipableReferenceDataBuilder creates a new builder instance.
func NewEquipableReferenceDataBuilder() *EquipableReferenceDataBuilder {
	return &EquipableReferenceDataBuilder{}
}

// Clone initializes the builder with data from the provided model.
func (b *EquipableReferenceDataBuilder) Clone(model EquipableReferenceData) *EquipableReferenceDataBuilder {
	*b = EquipableReferenceDataBuilder(model)
	return b
}

// Build assembles the final EquipableReferenceData from the builder.
func (b *EquipableReferenceDataBuilder) Build() EquipableReferenceData {
	return EquipableReferenceData{
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
		ownerId:        b.ownerId,
		flag:           b.flag,
		levelType:      b.levelType,
		level:          b.level,
		experience:     b.experience,
		hammersApplied: b.hammersApplied,
		expiration:     b.expiration,
	}
}

func (b *EquipableReferenceDataBuilder) SetStrength(value uint16) *EquipableReferenceDataBuilder {
	b.strength = value
	return b
}

// Setters for EquipableReferenceDataBuilder

func (b *EquipableReferenceDataBuilder) SetDexterity(value uint16) *EquipableReferenceDataBuilder {
	b.dexterity = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetIntelligence(value uint16) *EquipableReferenceDataBuilder {
	b.intelligence = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetLuck(value uint16) *EquipableReferenceDataBuilder {
	b.luck = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetHp(value uint16) *EquipableReferenceDataBuilder {
	b.hp = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetMp(value uint16) *EquipableReferenceDataBuilder {
	b.mp = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetWeaponAttack(value uint16) *EquipableReferenceDataBuilder {
	b.weaponAttack = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetMagicAttack(value uint16) *EquipableReferenceDataBuilder {
	b.magicAttack = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetWeaponDefense(value uint16) *EquipableReferenceDataBuilder {
	b.weaponDefense = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetMagicDefense(value uint16) *EquipableReferenceDataBuilder {
	b.magicDefense = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetAccuracy(value uint16) *EquipableReferenceDataBuilder {
	b.accuracy = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetAvoidability(value uint16) *EquipableReferenceDataBuilder {
	b.avoidability = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetHands(value uint16) *EquipableReferenceDataBuilder {
	b.hands = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetSpeed(value uint16) *EquipableReferenceDataBuilder {
	b.speed = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetJump(value uint16) *EquipableReferenceDataBuilder {
	b.jump = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetSlots(value uint16) *EquipableReferenceDataBuilder {
	b.slots = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetOwnerId(value uint32) *EquipableReferenceDataBuilder {
	b.ownerId = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetLocked(value bool) *EquipableReferenceDataBuilder {
	if value {
		b.flag = af.SetFlag(b.flag, af.FlagLock)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagLock)
	}
	return b
}

func (b *EquipableReferenceDataBuilder) SetSpikes(value bool) *EquipableReferenceDataBuilder {
	if value {
		b.flag = af.SetFlag(b.flag, af.FlagSpikes)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagSpikes)
	}
	return b
}

func (b *EquipableReferenceDataBuilder) SetKarmaUsed(value bool) *EquipableReferenceDataBuilder {
	if value {
		b.flag = af.SetFlag(b.flag, af.FlagKarmaEquip)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagKarmaEquip)
	}
	return b
}

func (b *EquipableReferenceDataBuilder) SetCold(value bool) *EquipableReferenceDataBuilder {
	if value {
		b.flag = af.SetFlag(b.flag, af.FlagCold)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagCold)
	}
	return b
}

func (b *EquipableReferenceDataBuilder) SetCanBeTraded(value bool) *EquipableReferenceDataBuilder {
	if value {
		b.flag = af.ClearFlag(b.flag, af.FlagUntradeable)
	} else {
		b.flag = af.SetFlag(b.flag, af.FlagUntradeable)
	}
	return b
}

func (b *EquipableReferenceDataBuilder) SetFlag(value uint16) *EquipableReferenceDataBuilder {
	b.flag = value
	return b
}

func (b *EquipableReferenceDataBuilder) AddFlag(f af.Flag) *EquipableReferenceDataBuilder {
	b.flag = af.SetFlag(b.flag, f)
	return b
}

func (b *EquipableReferenceDataBuilder) RemoveFlag(f af.Flag) *EquipableReferenceDataBuilder {
	b.flag = af.ClearFlag(b.flag, f)
	return b
}

func (b *EquipableReferenceDataBuilder) SetLevelType(value byte) *EquipableReferenceDataBuilder {
	b.levelType = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetLevel(value byte) *EquipableReferenceDataBuilder {
	b.level = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetExperience(value uint32) *EquipableReferenceDataBuilder {
	b.experience = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetHammersApplied(value uint32) *EquipableReferenceDataBuilder {
	b.hammersApplied = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetExpiration(value time.Time) *EquipableReferenceDataBuilder {
	b.expiration = value
	return b
}

type CashEquipableReferenceDataBuilder struct {
	cashId         uint64
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
	ownerId        uint32
	flag           uint16
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	expiration     time.Time
}

// NewCashEquipableReferenceDataBuilder creates a new builder instance.
func NewCashEquipableReferenceDataBuilder() *CashEquipableReferenceDataBuilder {
	return &CashEquipableReferenceDataBuilder{}
}

// Clone initializes the builder with data from the provided model.
func (b *CashEquipableReferenceDataBuilder) Clone(model CashEquipableReferenceData) *CashEquipableReferenceDataBuilder {
	*b = CashEquipableReferenceDataBuilder(model)
	return b
}

// Build assembles the final CashEquipableReferenceData from the builder.
func (b *CashEquipableReferenceDataBuilder) Build() CashEquipableReferenceData {
	return CashEquipableReferenceData{
		cashId:         b.cashId,
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
		ownerId:        b.ownerId,
		flag:           b.flag,
		levelType:      b.levelType,
		level:          b.level,
		experience:     b.experience,
		hammersApplied: b.hammersApplied,
		expiration:     b.expiration,
	}
}

func (b *CashEquipableReferenceDataBuilder) SetCashId(value uint64) *CashEquipableReferenceDataBuilder {
	b.cashId = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetStrength(value uint16) *CashEquipableReferenceDataBuilder {
	b.strength = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetDexterity(value uint16) *CashEquipableReferenceDataBuilder {
	b.dexterity = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetIntelligence(value uint16) *CashEquipableReferenceDataBuilder {
	b.intelligence = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetLuck(value uint16) *CashEquipableReferenceDataBuilder {
	b.luck = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetHp(value uint16) *CashEquipableReferenceDataBuilder {
	b.hp = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetMp(value uint16) *CashEquipableReferenceDataBuilder {
	b.mp = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetWeaponAttack(value uint16) *CashEquipableReferenceDataBuilder {
	b.weaponAttack = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetMagicAttack(value uint16) *CashEquipableReferenceDataBuilder {
	b.magicAttack = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetWeaponDefense(value uint16) *CashEquipableReferenceDataBuilder {
	b.weaponDefense = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetMagicDefense(value uint16) *CashEquipableReferenceDataBuilder {
	b.magicDefense = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetAccuracy(value uint16) *CashEquipableReferenceDataBuilder {
	b.accuracy = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetAvoidability(value uint16) *CashEquipableReferenceDataBuilder {
	b.avoidability = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetHands(value uint16) *CashEquipableReferenceDataBuilder {
	b.hands = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetSpeed(value uint16) *CashEquipableReferenceDataBuilder {
	b.speed = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetJump(value uint16) *CashEquipableReferenceDataBuilder {
	b.jump = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetSlots(value uint16) *CashEquipableReferenceDataBuilder {
	b.slots = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetOwnerId(value uint32) *CashEquipableReferenceDataBuilder {
	b.ownerId = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetLocked(value bool) *CashEquipableReferenceDataBuilder {
	if value {
		b.flag = af.SetFlag(b.flag, af.FlagLock)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagLock)
	}
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetSpikes(value bool) *CashEquipableReferenceDataBuilder {
	if value {
		b.flag = af.SetFlag(b.flag, af.FlagSpikes)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagSpikes)
	}
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetKarmaUsed(value bool) *CashEquipableReferenceDataBuilder {
	if value {
		b.flag = af.SetFlag(b.flag, af.FlagKarmaEquip)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagKarmaEquip)
	}
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetCold(value bool) *CashEquipableReferenceDataBuilder {
	if value {
		b.flag = af.SetFlag(b.flag, af.FlagCold)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagCold)
	}
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetCanBeTraded(value bool) *CashEquipableReferenceDataBuilder {
	if value {
		b.flag = af.ClearFlag(b.flag, af.FlagUntradeable)
	} else {
		b.flag = af.SetFlag(b.flag, af.FlagUntradeable)
	}
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetFlag(value uint16) *CashEquipableReferenceDataBuilder {
	b.flag = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) AddFlag(f af.Flag) *CashEquipableReferenceDataBuilder {
	b.flag = af.SetFlag(b.flag, f)
	return b
}

func (b *CashEquipableReferenceDataBuilder) RemoveFlag(f af.Flag) *CashEquipableReferenceDataBuilder {
	b.flag = af.ClearFlag(b.flag, f)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetLevelType(value byte) *CashEquipableReferenceDataBuilder {
	b.levelType = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetLevel(value byte) *CashEquipableReferenceDataBuilder {
	b.level = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetExperience(value uint32) *CashEquipableReferenceDataBuilder {
	b.experience = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetHammersApplied(value uint32) *CashEquipableReferenceDataBuilder {
	b.hammersApplied = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetExpiration(value time.Time) *CashEquipableReferenceDataBuilder {
	b.expiration = value
	return b
}

type ConsumableReferenceDataBuilder struct {
	quantity     uint32
	ownerId      uint32
	flag         uint16
	rechargeable uint64
}

func NewConsumableReferenceDataBuilder() *ConsumableReferenceDataBuilder {
	return &ConsumableReferenceDataBuilder{}
}

func (b *ConsumableReferenceDataBuilder) SetQuantity(value uint32) *ConsumableReferenceDataBuilder {
	b.quantity = value
	return b
}

func (b *ConsumableReferenceDataBuilder) SetOwnerId(value uint32) *ConsumableReferenceDataBuilder {
	b.ownerId = value
	return b
}

func (b *ConsumableReferenceDataBuilder) SetFlag(value uint16) *ConsumableReferenceDataBuilder {
	b.flag = value
	return b
}

func (b *ConsumableReferenceDataBuilder) SetRechargeable(value uint64) *ConsumableReferenceDataBuilder {
	b.rechargeable = value
	return b
}

func (b *ConsumableReferenceDataBuilder) Build() ConsumableReferenceData {
	return ConsumableReferenceData{
		quantity:     b.quantity,
		ownerId:      b.ownerId,
		flag:         b.flag,
		rechargeable: b.rechargeable,
	}
}

type SetupReferenceDataBuilder struct {
	quantity uint32
	ownerId  uint32
	flag     uint16
}

func NewSetupReferenceDataBuilder() *SetupReferenceDataBuilder {
	return &SetupReferenceDataBuilder{}
}

func (b *SetupReferenceDataBuilder) SetQuantity(value uint32) *SetupReferenceDataBuilder {
	b.quantity = value
	return b
}

func (b *SetupReferenceDataBuilder) SetOwnerId(value uint32) *SetupReferenceDataBuilder {
	b.ownerId = value
	return b
}

func (b *SetupReferenceDataBuilder) SetFlag(value uint16) *SetupReferenceDataBuilder {
	b.flag = value
	return b
}

func (b *SetupReferenceDataBuilder) Build() SetupReferenceData {
	return SetupReferenceData{
		quantity: b.quantity,
		ownerId:  b.ownerId,
		flag:     b.flag,
	}
}

type EtcReferenceDataBuilder struct {
	quantity uint32
	ownerId  uint32
	flag     uint16
}

func NewEtcReferenceDataBuilder() *EtcReferenceDataBuilder {
	return &EtcReferenceDataBuilder{}
}

func (b *EtcReferenceDataBuilder) SetQuantity(value uint32) *EtcReferenceDataBuilder {
	b.quantity = value
	return b
}

func (b *EtcReferenceDataBuilder) SetOwnerId(value uint32) *EtcReferenceDataBuilder {
	b.ownerId = value
	return b
}

func (b *EtcReferenceDataBuilder) SetFlag(value uint16) *EtcReferenceDataBuilder {
	b.flag = value
	return b
}

func (b *EtcReferenceDataBuilder) Build() EtcReferenceData {
	return EtcReferenceData{
		quantity: b.quantity,
		ownerId:  b.ownerId,
		flag:     b.flag,
	}
}

type CashReferenceDataBuilder struct {
	cashId     uint64
	quantity   uint32
	ownerId    uint32
	flag       uint16
	purchaseBy uint32
}

func NewCashReferenceDataBuilder() *CashReferenceDataBuilder {
	return &CashReferenceDataBuilder{}
}

func (b *CashReferenceDataBuilder) SetCashId(value uint64) *CashReferenceDataBuilder {
	b.cashId = value
	return b
}

func (b *CashReferenceDataBuilder) SetQuantity(value uint32) *CashReferenceDataBuilder {
	b.quantity = value
	return b
}

func (b *CashReferenceDataBuilder) SetOwnerId(value uint32) *CashReferenceDataBuilder {
	b.ownerId = value
	return b
}

func (b *CashReferenceDataBuilder) SetFlag(value uint16) *CashReferenceDataBuilder {
	b.flag = value
	return b
}

func (b *CashReferenceDataBuilder) SetPurchaseBy(value uint32) *CashReferenceDataBuilder {
	b.purchaseBy = value
	return b
}

func (b *CashReferenceDataBuilder) Build() CashReferenceData {
	return CashReferenceData{
		cashId:     b.cashId,
		quantity:   b.quantity,
		ownerId:    b.ownerId,
		flag:       b.flag,
		purchaseBy: b.purchaseBy,
	}
}

type PetReferenceDataBuilder struct {
	cashId        uint64
	ownerId       uint32
	flag          uint16
	purchaseBy    uint32
	name          string
	level         byte
	closeness     uint16
	fullness      byte
	expiration    time.Time
	slot          int8
	attribute     uint16
	skill         uint16
	remainingLife uint32
	attribute2    uint16
}

func NewPetReferenceDataBuilder() *PetReferenceDataBuilder {
	return &PetReferenceDataBuilder{}
}

func (b *PetReferenceDataBuilder) SetCashId(value uint64) *PetReferenceDataBuilder {
	b.cashId = value
	return b
}

func (b *PetReferenceDataBuilder) SetOwnerId(value uint32) *PetReferenceDataBuilder {
	b.ownerId = value
	return b
}

func (b *PetReferenceDataBuilder) SetFlag(value uint16) *PetReferenceDataBuilder {
	b.flag = value
	return b
}

func (b *PetReferenceDataBuilder) SetPurchaseBy(value uint32) *PetReferenceDataBuilder {
	b.purchaseBy = value
	return b
}

func (b *PetReferenceDataBuilder) SetName(value string) *PetReferenceDataBuilder {
	b.name = value
	return b
}

func (b *PetReferenceDataBuilder) SetLevel(value byte) *PetReferenceDataBuilder {
	b.level = value
	return b
}

func (b *PetReferenceDataBuilder) SetCloseness(value uint16) *PetReferenceDataBuilder {
	b.closeness = value
	return b
}

func (b *PetReferenceDataBuilder) SetFullness(value byte) *PetReferenceDataBuilder {
	b.fullness = value
	return b
}

func (b *PetReferenceDataBuilder) SetExpiration(value time.Time) *PetReferenceDataBuilder {
	b.expiration = value
	return b
}

func (b *PetReferenceDataBuilder) SetSlot(value int8) *PetReferenceDataBuilder {
	b.slot = value
	return b
}

func (b *PetReferenceDataBuilder) SetAttribute(value uint16) *PetReferenceDataBuilder {
	b.attribute = value
	return b
}

func (b *PetReferenceDataBuilder) SetSkill(value uint16) *PetReferenceDataBuilder {
	b.skill = value
	return b
}

func (b *PetReferenceDataBuilder) SetRemainingLife(value uint32) *PetReferenceDataBuilder {
	b.remainingLife = value
	return b
}

func (b *PetReferenceDataBuilder) SetAttribute2(value uint16) *PetReferenceDataBuilder {
	b.attribute2 = value
	return b
}

func (b *PetReferenceDataBuilder) Build() PetReferenceData {
	return PetReferenceData{
		cashId:        b.cashId,
		ownerId:       b.ownerId,
		flag:          b.flag,
		purchaseBy:    b.purchaseBy,
		name:          b.name,
		level:         b.level,
		closeness:     b.closeness,
		fullness:      b.fullness,
		expiration:    b.expiration,
		slot:          b.slot,
		attribute:     b.attribute,
		skill:         b.skill,
		remainingLife: b.remainingLife,
		attribute2:    b.attribute2,
	}
}
