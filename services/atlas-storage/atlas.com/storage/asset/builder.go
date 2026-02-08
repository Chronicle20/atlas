package asset

import (
	"time"

	"github.com/google/uuid"
)

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:             m.id,
		storageId:      m.storageId,
		slot:           m.slot,
		templateId:     m.templateId,
		expiration:     m.expiration,
		quantity:       m.quantity,
		ownerId:        m.ownerId,
		flag:           m.flag,
		rechargeable:   m.rechargeable,
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
		locked:         m.locked,
		spikes:         m.spikes,
		karmaUsed:      m.karmaUsed,
		cold:           m.cold,
		canBeTraded:    m.canBeTraded,
		levelType:      m.levelType,
		level:          m.level,
		experience:     m.experience,
		hammersApplied: m.hammersApplied,
		cashId:         m.cashId,
		commodityId:    m.commodityId,
		purchaseBy:     m.purchaseBy,
		petId:          m.petId,
	}
}

type ModelBuilder struct {
	id        uint32
	storageId uuid.UUID
	slot      int16
	templateId uint32
	expiration time.Time
	// stackable fields
	quantity     uint32
	ownerId      uint32
	flag         uint16
	rechargeable uint64
	// equipment fields
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
	locked         bool
	spikes         bool
	karmaUsed      bool
	cold           bool
	canBeTraded    bool
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	// cash fields
	cashId      int64
	commodityId uint32
	purchaseBy  uint32
	// pet reference
	petId uint32
}

func NewBuilder(storageId uuid.UUID, templateId uint32) *ModelBuilder {
	return &ModelBuilder{
		storageId:  storageId,
		templateId: templateId,
	}
}

func (b *ModelBuilder) SetId(id uint32) *ModelBuilder                  { b.id = id; return b }
func (b *ModelBuilder) SetStorageId(id uuid.UUID) *ModelBuilder        { b.storageId = id; return b }
func (b *ModelBuilder) SetSlot(slot int16) *ModelBuilder               { b.slot = slot; return b }
func (b *ModelBuilder) SetTemplateId(id uint32) *ModelBuilder          { b.templateId = id; return b }
func (b *ModelBuilder) SetExpiration(e time.Time) *ModelBuilder        { b.expiration = e; return b }
func (b *ModelBuilder) SetQuantity(q uint32) *ModelBuilder             { b.quantity = q; return b }
func (b *ModelBuilder) SetOwnerId(id uint32) *ModelBuilder             { b.ownerId = id; return b }
func (b *ModelBuilder) SetFlag(f uint16) *ModelBuilder                 { b.flag = f; return b }
func (b *ModelBuilder) SetRechargeable(r uint64) *ModelBuilder         { b.rechargeable = r; return b }
func (b *ModelBuilder) SetStrength(v uint16) *ModelBuilder             { b.strength = v; return b }
func (b *ModelBuilder) SetDexterity(v uint16) *ModelBuilder            { b.dexterity = v; return b }
func (b *ModelBuilder) SetIntelligence(v uint16) *ModelBuilder         { b.intelligence = v; return b }
func (b *ModelBuilder) SetLuck(v uint16) *ModelBuilder                 { b.luck = v; return b }
func (b *ModelBuilder) SetHp(v uint16) *ModelBuilder                   { b.hp = v; return b }
func (b *ModelBuilder) SetMp(v uint16) *ModelBuilder                   { b.mp = v; return b }
func (b *ModelBuilder) SetWeaponAttack(v uint16) *ModelBuilder         { b.weaponAttack = v; return b }
func (b *ModelBuilder) SetMagicAttack(v uint16) *ModelBuilder          { b.magicAttack = v; return b }
func (b *ModelBuilder) SetWeaponDefense(v uint16) *ModelBuilder        { b.weaponDefense = v; return b }
func (b *ModelBuilder) SetMagicDefense(v uint16) *ModelBuilder         { b.magicDefense = v; return b }
func (b *ModelBuilder) SetAccuracy(v uint16) *ModelBuilder             { b.accuracy = v; return b }
func (b *ModelBuilder) SetAvoidability(v uint16) *ModelBuilder         { b.avoidability = v; return b }
func (b *ModelBuilder) SetHands(v uint16) *ModelBuilder                { b.hands = v; return b }
func (b *ModelBuilder) SetSpeed(v uint16) *ModelBuilder                { b.speed = v; return b }
func (b *ModelBuilder) SetJump(v uint16) *ModelBuilder                 { b.jump = v; return b }
func (b *ModelBuilder) SetSlots(v uint16) *ModelBuilder                { b.slots = v; return b }
func (b *ModelBuilder) SetLocked(v bool) *ModelBuilder                 { b.locked = v; return b }
func (b *ModelBuilder) SetSpikes(v bool) *ModelBuilder                 { b.spikes = v; return b }
func (b *ModelBuilder) SetKarmaUsed(v bool) *ModelBuilder              { b.karmaUsed = v; return b }
func (b *ModelBuilder) SetCold(v bool) *ModelBuilder                   { b.cold = v; return b }
func (b *ModelBuilder) SetCanBeTraded(v bool) *ModelBuilder            { b.canBeTraded = v; return b }
func (b *ModelBuilder) SetLevelType(v byte) *ModelBuilder              { b.levelType = v; return b }
func (b *ModelBuilder) SetLevel(v byte) *ModelBuilder                  { b.level = v; return b }
func (b *ModelBuilder) SetExperience(v uint32) *ModelBuilder           { b.experience = v; return b }
func (b *ModelBuilder) SetHammersApplied(v uint32) *ModelBuilder       { b.hammersApplied = v; return b }
func (b *ModelBuilder) SetCashId(v int64) *ModelBuilder                { b.cashId = v; return b }
func (b *ModelBuilder) SetCommodityId(v uint32) *ModelBuilder          { b.commodityId = v; return b }
func (b *ModelBuilder) SetPurchaseBy(v uint32) *ModelBuilder           { b.purchaseBy = v; return b }
func (b *ModelBuilder) SetPetId(v uint32) *ModelBuilder                { b.petId = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{
		id:             b.id,
		storageId:      b.storageId,
		slot:           b.slot,
		templateId:     b.templateId,
		expiration:     b.expiration,
		quantity:       b.quantity,
		ownerId:        b.ownerId,
		flag:           b.flag,
		rechargeable:   b.rechargeable,
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
		locked:         b.locked,
		spikes:         b.spikes,
		karmaUsed:      b.karmaUsed,
		cold:           b.cold,
		canBeTraded:    b.canBeTraded,
		levelType:      b.levelType,
		level:          b.level,
		experience:     b.experience,
		hammersApplied: b.hammersApplied,
		cashId:         b.cashId,
		commodityId:    b.commodityId,
		purchaseBy:     b.purchaseBy,
		petId:          b.petId,
	}
}
