package asset

import "time"

type OwnerData struct {
	ownerId uint32
}

func (d OwnerData) OwnerId() uint32 { return d.ownerId }

type OwnerDataBuilder struct {
	ownerId uint32
}

func (b *OwnerDataBuilder) SetOwnerId(value uint32) *OwnerDataBuilder {
	b.ownerId = value
	return b
}

type StackableData struct {
	quantity uint32
}

func (d StackableData) Quantity() uint32 { return d.quantity }

type StackableDataBuilder struct {
	quantity uint32
}

func (b *StackableDataBuilder) SetQuantity(value uint32) *StackableDataBuilder {
	b.quantity = value
	return b
}

type FlagData struct {
	flag uint16
}

func (d FlagData) Flag() uint16 { return d.flag }

type FlagDataBuilder struct {
	flag uint16
}

func (b *FlagDataBuilder) SetFlag(value uint16) *FlagDataBuilder {
	b.flag = value
	return b
}

type CashData struct {
	cashId int64
}

func (d CashData) CashId() int64 { return d.cashId }

type CashDataBuilder struct {
	cashId int64
}

func (b *CashDataBuilder) SetCashId(value int64) *CashDataBuilder {
	b.cashId = value
	return b
}

type PurchaseData struct {
	purchaseBy uint32
}

func (d PurchaseData) PurchaseBy() uint32 { return d.purchaseBy }

type PurchaseDataBuilder struct {
	purchaseBy uint32
}

func (b *PurchaseDataBuilder) SetPurchaseBy(value uint32) *PurchaseDataBuilder {
	b.purchaseBy = value
	return b
}

type StatisticData struct {
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
}

func (e StatisticData) Strength() uint16      { return e.strength }
func (e StatisticData) Dexterity() uint16     { return e.dexterity }
func (e StatisticData) Intelligence() uint16  { return e.intelligence }
func (e StatisticData) Luck() uint16          { return e.luck }
func (e StatisticData) HP() uint16            { return e.hp }
func (e StatisticData) MP() uint16            { return e.mp }
func (e StatisticData) WeaponAttack() uint16  { return e.weaponAttack }
func (e StatisticData) MagicAttack() uint16   { return e.magicAttack }
func (e StatisticData) WeaponDefense() uint16 { return e.weaponDefense }
func (e StatisticData) MagicDefense() uint16  { return e.magicDefense }
func (e StatisticData) Accuracy() uint16      { return e.accuracy }
func (e StatisticData) Avoidability() uint16  { return e.avoidability }
func (e StatisticData) Hands() uint16         { return e.hands }
func (e StatisticData) Speed() uint16         { return e.speed }
func (e StatisticData) Jump() uint16          { return e.jump }

type StatisticDataBuilder struct {
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
}

func (b *StatisticDataBuilder) SetStrength(value uint16) *StatisticDataBuilder {
	b.strength = value
	return b
}

func (b *StatisticDataBuilder) SetDexterity(value uint16) *StatisticDataBuilder {
	b.dexterity = value
	return b
}

func (b *StatisticDataBuilder) SetIntelligence(value uint16) *StatisticDataBuilder {
	b.intelligence = value
	return b
}

func (b *StatisticDataBuilder) SetLuck(value uint16) *StatisticDataBuilder {
	b.luck = value
	return b
}

func (b *StatisticDataBuilder) SetHp(value uint16) *StatisticDataBuilder {
	b.hp = value
	return b
}

func (b *StatisticDataBuilder) SetMp(value uint16) *StatisticDataBuilder {
	b.mp = value
	return b
}

func (b *StatisticDataBuilder) SetWeaponAttack(value uint16) *StatisticDataBuilder {
	b.weaponAttack = value
	return b
}

func (b *StatisticDataBuilder) SetMagicAttack(value uint16) *StatisticDataBuilder {
	b.magicAttack = value
	return b
}

func (b *StatisticDataBuilder) SetWeaponDefense(value uint16) *StatisticDataBuilder {
	b.weaponDefense = value
	return b
}

func (b *StatisticDataBuilder) SetMagicDefense(value uint16) *StatisticDataBuilder {
	b.magicDefense = value
	return b
}

func (b *StatisticDataBuilder) SetAccuracy(value uint16) *StatisticDataBuilder {
	b.accuracy = value
	return b
}

func (b *StatisticDataBuilder) SetAvoidability(value uint16) *StatisticDataBuilder {
	b.avoidability = value
	return b
}

func (b *StatisticDataBuilder) SetHands(value uint16) *StatisticDataBuilder {
	b.hands = value
	return b
}

func (b *StatisticDataBuilder) SetSpeed(value uint16) *StatisticDataBuilder {
	b.speed = value
	return b
}

func (b *StatisticDataBuilder) SetJump(value uint16) *StatisticDataBuilder {
	b.jump = value
	return b
}

type EquipableReferenceData struct {
	StatisticData
	OwnerData
	slots          uint16
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

func (e EquipableReferenceData) Slots() uint16          { return e.slots }
func (e EquipableReferenceData) IsLocked() bool         { return e.locked }
func (e EquipableReferenceData) HasSpikes() bool        { return e.spikes }
func (e EquipableReferenceData) IsKarmaUsed() bool      { return e.karmaUsed }
func (e EquipableReferenceData) IsCold() bool           { return e.cold }
func (e EquipableReferenceData) CanBeTraded() bool      { return e.canBeTraded }
func (e EquipableReferenceData) LevelType() byte        { return e.levelType }
func (e EquipableReferenceData) Level() byte            { return e.level }
func (e EquipableReferenceData) Experience() uint32     { return e.experience }
func (e EquipableReferenceData) HammersApplied() uint32 { return e.hammersApplied }
func (e EquipableReferenceData) Expiration() time.Time  { return e.expiration }

type EquipableReferenceDataBuilder struct {
	StatisticDataBuilder
	OwnerDataBuilder
	slots          uint16
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

// NewEquipableReferenceDataBuilder creates a new builder instance.
func NewEquipableReferenceDataBuilder() *EquipableReferenceDataBuilder {
	return &EquipableReferenceDataBuilder{}
}

// Clone initializes the builder with data from the provided model.
func (b *EquipableReferenceDataBuilder) Clone(model EquipableReferenceData) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder = StatisticDataBuilder{
		strength:      model.strength,
		dexterity:     model.dexterity,
		intelligence:  model.intelligence,
		luck:          model.luck,
		hp:            model.hp,
		mp:            model.mp,
		weaponAttack:  model.weaponAttack,
		magicAttack:   model.magicAttack,
		weaponDefense: model.weaponDefense,
		magicDefense:  model.magicDefense,
		accuracy:      model.accuracy,
		avoidability:  model.avoidability,
		hands:         model.hands,
		speed:         model.speed,
		jump:          model.jump,
	}
	b.OwnerDataBuilder = OwnerDataBuilder{
		ownerId: model.ownerId,
	}
	b.slots = model.slots
	b.locked = model.locked
	b.spikes = model.spikes
	b.karmaUsed = model.karmaUsed
	b.cold = model.cold
	b.canBeTraded = model.canBeTraded
	b.levelType = model.levelType
	b.level = model.level
	b.experience = model.experience
	b.hammersApplied = model.hammersApplied
	b.expiration = model.expiration
	return b
}

// Build assembles the final EquipableReferenceData from the builder.
func (b *EquipableReferenceDataBuilder) Build() EquipableReferenceData {
	return EquipableReferenceData{
		StatisticData: StatisticData{
			strength:      b.StatisticDataBuilder.strength,
			dexterity:     b.StatisticDataBuilder.dexterity,
			intelligence:  b.StatisticDataBuilder.intelligence,
			luck:          b.StatisticDataBuilder.luck,
			hp:            b.StatisticDataBuilder.hp,
			mp:            b.StatisticDataBuilder.mp,
			weaponAttack:  b.StatisticDataBuilder.weaponAttack,
			magicAttack:   b.StatisticDataBuilder.magicAttack,
			weaponDefense: b.StatisticDataBuilder.weaponDefense,
			magicDefense:  b.StatisticDataBuilder.magicDefense,
			accuracy:      b.StatisticDataBuilder.accuracy,
			avoidability:  b.StatisticDataBuilder.avoidability,
			hands:         b.StatisticDataBuilder.hands,
			speed:         b.StatisticDataBuilder.speed,
			jump:          b.StatisticDataBuilder.jump,
		},
		OwnerData: OwnerData{
			ownerId: b.OwnerDataBuilder.ownerId,
		},
		slots:          b.slots,
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

func (b *EquipableReferenceDataBuilder) SetStrength(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetStrength(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetDexterity(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetDexterity(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetIntelligence(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetIntelligence(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetLuck(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetLuck(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetHp(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetHp(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetMp(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetMp(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetWeaponAttack(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetWeaponAttack(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetMagicAttack(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetMagicAttack(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetWeaponDefense(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetWeaponDefense(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetMagicDefense(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetMagicDefense(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetAccuracy(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetAccuracy(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetAvoidability(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetAvoidability(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetHands(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetHands(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetSpeed(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetSpeed(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetJump(value uint16) *EquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetJump(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetSlots(value uint16) *EquipableReferenceDataBuilder {
	b.slots = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetOwnerId(value uint32) *EquipableReferenceDataBuilder {
	b.OwnerDataBuilder.SetOwnerId(value)
	return b
}

func (b *EquipableReferenceDataBuilder) SetLocked(value bool) *EquipableReferenceDataBuilder {
	b.locked = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetSpikes(value bool) *EquipableReferenceDataBuilder {
	b.spikes = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetKarmaUsed(value bool) *EquipableReferenceDataBuilder {
	b.karmaUsed = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetCold(value bool) *EquipableReferenceDataBuilder {
	b.cold = value
	return b
}

func (b *EquipableReferenceDataBuilder) SetCanBeTraded(value bool) *EquipableReferenceDataBuilder {
	b.canBeTraded = value
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

type CashEquipableReferenceData struct {
	StatisticData
	OwnerData
	CashData
	slots          uint16
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

func (e CashEquipableReferenceData) GetSlots() uint16          { return e.slots }
func (e CashEquipableReferenceData) IsLocked() bool            { return e.locked }
func (e CashEquipableReferenceData) HasSpikes() bool           { return e.spikes }
func (e CashEquipableReferenceData) IsKarmaUsed() bool         { return e.karmaUsed }
func (e CashEquipableReferenceData) IsCold() bool              { return e.cold }
func (e CashEquipableReferenceData) CanBeTraded() bool         { return e.canBeTraded }
func (e CashEquipableReferenceData) GetLevelType() byte        { return e.levelType }
func (e CashEquipableReferenceData) GetLevel() byte            { return e.level }
func (e CashEquipableReferenceData) GetExperience() uint32     { return e.experience }
func (e CashEquipableReferenceData) GetHammersApplied() uint32 { return e.hammersApplied }
func (e CashEquipableReferenceData) GetExpiration() time.Time  { return e.expiration }

type CashEquipableReferenceDataBuilder struct {
	StatisticDataBuilder
	OwnerDataBuilder
	CashDataBuilder
	slots          uint16
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

// NewCashEquipableReferenceDataBuilder creates a new builder instance.
func NewCashEquipableReferenceDataBuilder() *CashEquipableReferenceDataBuilder {
	return &CashEquipableReferenceDataBuilder{}
}

// Clone initializes the builder with data from the provided model.
func (b *CashEquipableReferenceDataBuilder) Clone(model CashEquipableReferenceData) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder = StatisticDataBuilder{
		strength:      model.strength,
		dexterity:     model.dexterity,
		intelligence:  model.intelligence,
		luck:          model.luck,
		hp:            model.hp,
		mp:            model.mp,
		weaponAttack:  model.weaponAttack,
		magicAttack:   model.magicAttack,
		weaponDefense: model.weaponDefense,
		magicDefense:  model.magicDefense,
		accuracy:      model.accuracy,
		avoidability:  model.avoidability,
		hands:         model.hands,
		speed:         model.speed,
		jump:          model.jump,
	}
	b.OwnerDataBuilder = OwnerDataBuilder{
		ownerId: model.ownerId,
	}
	b.CashDataBuilder = CashDataBuilder{
		cashId: model.cashId,
	}
	b.slots = model.slots
	b.locked = model.locked
	b.spikes = model.spikes
	b.karmaUsed = model.karmaUsed
	b.cold = model.cold
	b.canBeTraded = model.canBeTraded
	b.levelType = model.levelType
	b.level = model.level
	b.experience = model.experience
	b.hammersApplied = model.hammersApplied
	b.expiration = model.expiration
	return b
}

// Build assembles the final CashEquipableReferenceData from the builder.
func (b *CashEquipableReferenceDataBuilder) Build() CashEquipableReferenceData {
	return CashEquipableReferenceData{
		StatisticData: StatisticData{
			strength:      b.StatisticDataBuilder.strength,
			dexterity:     b.StatisticDataBuilder.dexterity,
			intelligence:  b.StatisticDataBuilder.intelligence,
			luck:          b.StatisticDataBuilder.luck,
			hp:            b.StatisticDataBuilder.hp,
			mp:            b.StatisticDataBuilder.mp,
			weaponAttack:  b.StatisticDataBuilder.weaponAttack,
			magicAttack:   b.StatisticDataBuilder.magicAttack,
			weaponDefense: b.StatisticDataBuilder.weaponDefense,
			magicDefense:  b.StatisticDataBuilder.magicDefense,
			accuracy:      b.StatisticDataBuilder.accuracy,
			avoidability:  b.StatisticDataBuilder.avoidability,
			hands:         b.StatisticDataBuilder.hands,
			speed:         b.StatisticDataBuilder.speed,
			jump:          b.StatisticDataBuilder.jump,
		},
		OwnerData: OwnerData{
			ownerId: b.OwnerDataBuilder.ownerId,
		},
		CashData: CashData{
			cashId: b.CashDataBuilder.cashId,
		},
		slots:          b.slots,
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

func (b *CashEquipableReferenceDataBuilder) SetCashId(value int64) *CashEquipableReferenceDataBuilder {
	b.CashDataBuilder.SetCashId(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetStrength(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetStrength(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetDexterity(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetDexterity(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetIntelligence(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetIntelligence(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetLuck(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetLuck(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetHp(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetHp(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetMp(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetMp(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetWeaponAttack(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetWeaponAttack(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetMagicAttack(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetMagicAttack(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetWeaponDefense(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetWeaponDefense(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetMagicDefense(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetMagicDefense(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetAccuracy(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetAccuracy(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetAvoidability(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetAvoidability(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetHands(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetHands(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetSpeed(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetSpeed(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetJump(value uint16) *CashEquipableReferenceDataBuilder {
	b.StatisticDataBuilder.SetJump(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetSlots(value uint16) *CashEquipableReferenceDataBuilder {
	b.slots = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetOwnerId(value uint32) *CashEquipableReferenceDataBuilder {
	b.OwnerDataBuilder.SetOwnerId(value)
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetLocked(value bool) *CashEquipableReferenceDataBuilder {
	b.locked = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetSpikes(value bool) *CashEquipableReferenceDataBuilder {
	b.spikes = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetKarmaUsed(value bool) *CashEquipableReferenceDataBuilder {
	b.karmaUsed = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetCold(value bool) *CashEquipableReferenceDataBuilder {
	b.cold = value
	return b
}

func (b *CashEquipableReferenceDataBuilder) SetCanBeTraded(value bool) *CashEquipableReferenceDataBuilder {
	b.canBeTraded = value
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

type ConsumableReferenceData struct {
	OwnerData
	StackableData
	FlagData
	rechargeable uint64
}

func (c ConsumableReferenceData) Rechargeable() uint64 {
	return c.rechargeable
}

type ConsumableReferenceDataBuilder struct {
	OwnerDataBuilder
	StackableDataBuilder
	FlagDataBuilder
	rechargeable uint64
}

func NewConsumableReferenceDataBuilder() *ConsumableReferenceDataBuilder {
	return &ConsumableReferenceDataBuilder{}
}

func (b *ConsumableReferenceDataBuilder) SetQuantity(value uint32) *ConsumableReferenceDataBuilder {
	b.StackableDataBuilder.SetQuantity(value)
	return b
}

func (b *ConsumableReferenceDataBuilder) SetOwnerId(value uint32) *ConsumableReferenceDataBuilder {
	b.OwnerDataBuilder.SetOwnerId(value)
	return b
}

func (b *ConsumableReferenceDataBuilder) SetFlag(value uint16) *ConsumableReferenceDataBuilder {
	b.FlagDataBuilder.SetFlag(value)
	return b
}

func (b *ConsumableReferenceDataBuilder) SetRechargeable(value uint64) *ConsumableReferenceDataBuilder {
	b.rechargeable = value
	return b
}

func (b *ConsumableReferenceDataBuilder) Build() ConsumableReferenceData {
	return ConsumableReferenceData{
		OwnerData: OwnerData{
			ownerId: b.OwnerDataBuilder.ownerId,
		},
		StackableData: StackableData{
			quantity: b.StackableDataBuilder.quantity,
		},
		FlagData: FlagData{
			flag: b.FlagDataBuilder.flag,
		},
		rechargeable: b.rechargeable,
	}
}

type SetupReferenceData struct {
	OwnerData
	StackableData
	FlagData
}

type SetupReferenceDataBuilder struct {
	OwnerDataBuilder
	StackableDataBuilder
	FlagDataBuilder
}

func NewSetupReferenceDataBuilder() *SetupReferenceDataBuilder {
	return &SetupReferenceDataBuilder{}
}

func (b *SetupReferenceDataBuilder) SetQuantity(value uint32) *SetupReferenceDataBuilder {
	b.StackableDataBuilder.SetQuantity(value)
	return b
}

func (b *SetupReferenceDataBuilder) SetOwnerId(value uint32) *SetupReferenceDataBuilder {
	b.OwnerDataBuilder.SetOwnerId(value)
	return b
}

func (b *SetupReferenceDataBuilder) SetFlag(value uint16) *SetupReferenceDataBuilder {
	b.FlagDataBuilder.SetFlag(value)
	return b
}

func (b *SetupReferenceDataBuilder) Build() SetupReferenceData {
	return SetupReferenceData{
		OwnerData: OwnerData{
			ownerId: b.OwnerDataBuilder.ownerId,
		},
		StackableData: StackableData{
			quantity: b.StackableDataBuilder.quantity,
		},
		FlagData: FlagData{
			flag: b.FlagDataBuilder.flag,
		},
	}
}

type EtcReferenceData struct {
	OwnerData
	StackableData
	FlagData
}

type EtcReferenceDataBuilder struct {
	OwnerDataBuilder
	StackableDataBuilder
	FlagDataBuilder
}

func NewEtcReferenceDataBuilder() *EtcReferenceDataBuilder {
	return &EtcReferenceDataBuilder{}
}

func (b *EtcReferenceDataBuilder) SetQuantity(value uint32) *EtcReferenceDataBuilder {
	b.StackableDataBuilder.SetQuantity(value)
	return b
}

func (b *EtcReferenceDataBuilder) SetOwnerId(value uint32) *EtcReferenceDataBuilder {
	b.OwnerDataBuilder.SetOwnerId(value)
	return b
}

func (b *EtcReferenceDataBuilder) SetFlag(value uint16) *EtcReferenceDataBuilder {
	b.FlagDataBuilder.SetFlag(value)
	return b
}

func (b *EtcReferenceDataBuilder) Build() EtcReferenceData {
	return EtcReferenceData{
		OwnerData: OwnerData{
			ownerId: b.OwnerDataBuilder.ownerId,
		},
		StackableData: StackableData{
			quantity: b.StackableDataBuilder.quantity,
		},
		FlagData: FlagData{
			flag: b.FlagDataBuilder.flag,
		},
	}
}

type CashReferenceData struct {
	OwnerData
	StackableData
	FlagData
	CashData
	PurchaseData
}

type CashReferenceDataBuilder struct {
	OwnerDataBuilder
	StackableDataBuilder
	FlagDataBuilder
	CashDataBuilder
	PurchaseDataBuilder
}

func NewCashReferenceDataBuilder() *CashReferenceDataBuilder {
	return &CashReferenceDataBuilder{}
}

func (b *CashReferenceDataBuilder) SetCashId(value int64) *CashReferenceDataBuilder {
	b.CashDataBuilder.SetCashId(value)
	return b
}

func (b *CashReferenceDataBuilder) SetQuantity(value uint32) *CashReferenceDataBuilder {
	b.StackableDataBuilder.SetQuantity(value)
	return b
}

func (b *CashReferenceDataBuilder) SetOwnerId(value uint32) *CashReferenceDataBuilder {
	b.OwnerDataBuilder.SetOwnerId(value)
	return b
}

func (b *CashReferenceDataBuilder) SetFlag(value uint16) *CashReferenceDataBuilder {
	b.FlagDataBuilder.SetFlag(value)
	return b
}

func (b *CashReferenceDataBuilder) SetPurchaseBy(value uint32) *CashReferenceDataBuilder {
	b.PurchaseDataBuilder.SetPurchaseBy(value)
	return b
}

func (b *CashReferenceDataBuilder) Build() CashReferenceData {
	return CashReferenceData{
		OwnerData: OwnerData{
			ownerId: b.OwnerDataBuilder.ownerId,
		},
		StackableData: StackableData{
			quantity: b.StackableDataBuilder.quantity,
		},
		FlagData: FlagData{
			flag: b.FlagDataBuilder.flag,
		},
		CashData: CashData{
			cashId: b.CashDataBuilder.cashId,
		},
		PurchaseData: PurchaseData{
			purchaseBy: b.PurchaseDataBuilder.purchaseBy,
		},
	}
}

type PetReferenceData struct {
	OwnerData
	FlagData
	CashData
	PurchaseData
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

func (d PetReferenceData) Name() string {
	return d.name
}

func (d PetReferenceData) Level() byte {
	return d.level
}

func (d PetReferenceData) Closeness() uint16 {
	return d.closeness
}

func (d PetReferenceData) Fullness() byte {
	return d.fullness
}

func (d PetReferenceData) Slot() int8 {
	return d.slot
}

type PetReferenceDataBuilder struct {
	OwnerDataBuilder
	FlagDataBuilder
	CashDataBuilder
	PurchaseDataBuilder
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

func (b *PetReferenceDataBuilder) SetCashId(value int64) *PetReferenceDataBuilder {
	b.CashDataBuilder.SetCashId(value)
	return b
}

func (b *PetReferenceDataBuilder) SetOwnerId(value uint32) *PetReferenceDataBuilder {
	b.OwnerDataBuilder.SetOwnerId(value)
	return b
}

func (b *PetReferenceDataBuilder) SetFlag(value uint16) *PetReferenceDataBuilder {
	b.FlagDataBuilder.SetFlag(value)
	return b
}

func (b *PetReferenceDataBuilder) SetPurchaseBy(value uint32) *PetReferenceDataBuilder {
	b.PurchaseDataBuilder.SetPurchaseBy(value)
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
		OwnerData: OwnerData{
			ownerId: b.OwnerDataBuilder.ownerId,
		},
		FlagData: FlagData{
			flag: b.FlagDataBuilder.flag,
		},
		CashData: CashData{
			cashId: b.CashDataBuilder.cashId,
		},
		PurchaseData: PurchaseData{
			purchaseBy: b.PurchaseDataBuilder.purchaseBy,
		},
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
