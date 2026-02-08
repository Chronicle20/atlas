package asset

import (
	"time"

	"github.com/google/uuid"
)

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:             m.id,
		compartmentId:  m.compartmentId,
		slot:           m.slot,
		templateId:     m.templateId,
		expiration:     m.expiration,
		createdAt:      m.createdAt,
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
		equippedSince:  m.equippedSince,
		cashId:         m.cashId,
		commodityId:    m.commodityId,
		purchaseBy:     m.purchaseBy,
		petId:          m.petId,
	}
}

type ModelBuilder struct {
	id            uint32
	compartmentId uuid.UUID
	slot          int16
	templateId    uint32
	expiration    time.Time
	createdAt     time.Time
	quantity      uint32
	ownerId       uint32
	flag          uint16
	rechargeable  uint64
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
	equippedSince  *time.Time
	cashId         int64
	commodityId    uint32
	purchaseBy     uint32
	petId          uint32
}

func NewBuilder(compartmentId uuid.UUID, templateId uint32) *ModelBuilder {
	return &ModelBuilder{
		compartmentId: compartmentId,
		templateId:    templateId,
	}
}

func (b *ModelBuilder) SetId(id uint32) *ModelBuilder               { b.id = id; return b }
func (b *ModelBuilder) SetSlot(slot int16) *ModelBuilder            { b.slot = slot; return b }
func (b *ModelBuilder) SetExpiration(e time.Time) *ModelBuilder     { b.expiration = e; return b }
func (b *ModelBuilder) SetPetId(v uint32) *ModelBuilder             { b.petId = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{
		id:             b.id,
		compartmentId:  b.compartmentId,
		slot:           b.slot,
		templateId:     b.templateId,
		expiration:     b.expiration,
		createdAt:      b.createdAt,
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
		equippedSince:  b.equippedSince,
		cashId:         b.cashId,
		commodityId:    b.commodityId,
		purchaseBy:     b.purchaseBy,
		petId:          b.petId,
	}
}
