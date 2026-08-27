package asset

import (
	"time"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

type EquipableReferenceData struct {
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

func (e EquipableReferenceData) GetStrength() uint16      { return e.strength }
func (e EquipableReferenceData) GetDexterity() uint16     { return e.dexterity }
func (e EquipableReferenceData) GetIntelligence() uint16  { return e.intelligence }
func (e EquipableReferenceData) GetLuck() uint16          { return e.luck }
func (e EquipableReferenceData) GetHP() uint16            { return e.hp }
func (e EquipableReferenceData) GetMP() uint16            { return e.mp }
func (e EquipableReferenceData) GetWeaponAttack() uint16  { return e.weaponAttack }
func (e EquipableReferenceData) GetMagicAttack() uint16   { return e.magicAttack }
func (e EquipableReferenceData) GetWeaponDefense() uint16 { return e.weaponDefense }
func (e EquipableReferenceData) GetMagicDefense() uint16  { return e.magicDefense }
func (e EquipableReferenceData) GetAccuracy() uint16      { return e.accuracy }
func (e EquipableReferenceData) GetAvoidability() uint16  { return e.avoidability }
func (e EquipableReferenceData) GetHands() uint16         { return e.hands }
func (e EquipableReferenceData) GetSpeed() uint16         { return e.speed }
func (e EquipableReferenceData) GetJump() uint16          { return e.jump }
func (e EquipableReferenceData) GetSlots() uint16         { return e.slots }
func (e EquipableReferenceData) GetOwnerId() uint32       { return e.ownerId }
func (e EquipableReferenceData) IsLocked() bool           { return af.HasFlag(e.flag, af.FlagLock) }
func (e EquipableReferenceData) HasSpikes() bool          { return af.HasFlag(e.flag, af.FlagSpikes) }

// IsKarmaUsed reads the EQUIP karma bit (0x10) unconditionally. Unlike the six
// asset models, EquipableReferenceData carries no template id — it is the
// equip-shaped reference block hanging off an asset that holds the id — but it
// is equip-class BY TYPE, so the bit is fixed rather than resolved. The setter
// (SetKarmaUsed) already writes FlagKarmaEquip; before task-223 this getter read
// FlagKarmaUse (0x02, the BUNDLE bit) and the pair never round-tripped.
func (e EquipableReferenceData) IsKarmaUsed() bool { return af.HasFlag(e.flag, af.FlagKarmaEquip) }

func (e EquipableReferenceData) IsCold() bool { return af.HasFlag(e.flag, af.FlagCold) }

func (e EquipableReferenceData) CanBeTraded() bool         { return !af.HasFlag(e.flag, af.FlagUntradeable) }
func (e EquipableReferenceData) GetLevelType() byte        { return e.levelType }
func (e EquipableReferenceData) GetLevel() byte            { return e.level }
func (e EquipableReferenceData) GetExperience() uint32     { return e.experience }
func (e EquipableReferenceData) GetHammersApplied() uint32 { return e.hammersApplied }
func (e EquipableReferenceData) GetExpiration() time.Time  { return e.expiration }
func (e EquipableReferenceData) Flags() uint16             { return e.flag }

type CashEquipableReferenceData struct {
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

func (e CashEquipableReferenceData) GetCashId() uint64        { return e.cashId }
func (e CashEquipableReferenceData) GetStrength() uint16      { return e.strength }
func (e CashEquipableReferenceData) GetDexterity() uint16     { return e.dexterity }
func (e CashEquipableReferenceData) GetIntelligence() uint16  { return e.intelligence }
func (e CashEquipableReferenceData) GetLuck() uint16          { return e.luck }
func (e CashEquipableReferenceData) GetHP() uint16            { return e.hp }
func (e CashEquipableReferenceData) GetMP() uint16            { return e.mp }
func (e CashEquipableReferenceData) GetWeaponAttack() uint16  { return e.weaponAttack }
func (e CashEquipableReferenceData) GetMagicAttack() uint16   { return e.magicAttack }
func (e CashEquipableReferenceData) GetWeaponDefense() uint16 { return e.weaponDefense }
func (e CashEquipableReferenceData) GetMagicDefense() uint16  { return e.magicDefense }
func (e CashEquipableReferenceData) GetAccuracy() uint16      { return e.accuracy }
func (e CashEquipableReferenceData) GetAvoidability() uint16  { return e.avoidability }
func (e CashEquipableReferenceData) GetHands() uint16         { return e.hands }
func (e CashEquipableReferenceData) GetSpeed() uint16         { return e.speed }
func (e CashEquipableReferenceData) GetJump() uint16          { return e.jump }
func (e CashEquipableReferenceData) GetSlots() uint16         { return e.slots }
func (e CashEquipableReferenceData) GetOwnerId() uint32       { return e.ownerId }
func (e CashEquipableReferenceData) IsLocked() bool           { return af.HasFlag(e.flag, af.FlagLock) }

func (e CashEquipableReferenceData) HasSpikes() bool { return af.HasFlag(e.flag, af.FlagSpikes) }

// See EquipableReferenceData.IsKarmaUsed: equip-class by type, so the bit is fixed.
func (e CashEquipableReferenceData) IsKarmaUsed() bool { return af.HasFlag(e.flag, af.FlagKarmaEquip) }
func (e CashEquipableReferenceData) IsCold() bool      { return af.HasFlag(e.flag, af.FlagCold) }
func (e CashEquipableReferenceData) CanBeTraded() bool {
	return !af.HasFlag(e.flag, af.FlagUntradeable)
}
func (e CashEquipableReferenceData) Flags() uint16             { return e.flag }
func (e CashEquipableReferenceData) GetLevelType() byte        { return e.levelType }
func (e CashEquipableReferenceData) GetLevel() byte            { return e.level }
func (e CashEquipableReferenceData) GetExperience() uint32     { return e.experience }
func (e CashEquipableReferenceData) GetHammersApplied() uint32 { return e.hammersApplied }
func (e CashEquipableReferenceData) GetExpiration() time.Time  { return e.expiration }

type ConsumableReferenceData struct {
	quantity     uint32
	ownerId      uint32
	flag         uint16
	rechargeable uint64
}

func (c ConsumableReferenceData) Quantity() uint32 {
	return c.quantity
}

func (c ConsumableReferenceData) Flag() uint16 {
	return c.flag
}

func (c ConsumableReferenceData) Rechargeable() uint64 {
	return c.rechargeable
}

type SetupReferenceData struct {
	quantity uint32
	ownerId  uint32
	flag     uint16
}

func (c SetupReferenceData) Quantity() uint32 {
	return c.quantity
}

func (c SetupReferenceData) Flag() uint16 {
	return c.flag
}

type EtcReferenceData struct {
	quantity uint32
	ownerId  uint32
	flag     uint16
}

func (c EtcReferenceData) Quantity() uint32 {
	return c.quantity
}

func (c EtcReferenceData) Flag() uint16 {
	return c.flag
}

type CashReferenceData struct {
	cashId     uint64
	quantity   uint32
	ownerId    uint32
	flag       uint16
	purchaseBy uint32
}

func (c CashReferenceData) Quantity() uint32 {
	return c.quantity
}

func (c CashReferenceData) CashId() uint64 {
	return c.cashId
}

func (c CashReferenceData) OwnerId() uint32 {
	return c.ownerId
}

func (c CashReferenceData) Flag() uint16 {
	return c.flag
}

type PetReferenceData struct {
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

func (d PetReferenceData) CashId() uint64 {
	return d.cashId
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
