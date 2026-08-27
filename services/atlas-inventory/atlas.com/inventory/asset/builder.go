package asset

import (
	"time"

	"github.com/google/uuid"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

func Clone(m Model) *Builder {
	return &Builder{
		id:             m.id,
		compartmentId:  m.compartmentId,
		slot:           m.slot,
		templateId:     m.templateId,
		expiration:     m.expiration,
		createdAt:      m.createdAt,
		quantity:       m.quantity,
		ownerId:        m.ownerId,
		owner:          m.owner,
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

type Builder struct {
	id            uint32
	compartmentId uuid.UUID
	slot          int16
	templateId    uint32
	expiration    time.Time
	createdAt     time.Time
	// stackable fields
	quantity     uint32
	ownerId      uint32
	owner        string
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
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	equippedSince  *time.Time
	// cash fields
	cashId      int64
	commodityId uint32
	purchaseBy  uint32
	// pet reference
	petId uint32
}

func NewBuilder(compartmentId uuid.UUID, templateId uint32) *Builder {
	return &Builder{
		compartmentId: compartmentId,
		templateId:    templateId,
	}
}

func (b *Builder) SetId(id uint32) *Builder               { b.id = id; return b }
func (b *Builder) SetCompartmentId(id uuid.UUID) *Builder { b.compartmentId = id; return b }
func (b *Builder) SetSlot(slot int16) *Builder            { b.slot = slot; return b }
func (b *Builder) SetTemplateId(id uint32) *Builder       { b.templateId = id; return b }
func (b *Builder) SetExpiration(e time.Time) *Builder     { b.expiration = e; return b }
func (b *Builder) SetCreatedAt(t time.Time) *Builder      { b.createdAt = t; return b }
func (b *Builder) SetQuantity(q uint32) *Builder          { b.quantity = q; return b }
func (b *Builder) SetOwnerId(id uint32) *Builder          { b.ownerId = id; return b }
func (b *Builder) SetOwner(o string) *Builder             { b.owner = o; return b }
func (b *Builder) SetFlag(f uint16) *Builder              { b.flag = f; return b }
func (b *Builder) SetRechargeable(r uint64) *Builder      { b.rechargeable = r; return b }
func (b *Builder) SetStrength(v uint16) *Builder          { b.strength = v; return b }
func (b *Builder) SetDexterity(v uint16) *Builder         { b.dexterity = v; return b }
func (b *Builder) SetIntelligence(v uint16) *Builder      { b.intelligence = v; return b }
func (b *Builder) SetLuck(v uint16) *Builder              { b.luck = v; return b }
func (b *Builder) SetHp(v uint16) *Builder                { b.hp = v; return b }
func (b *Builder) SetMp(v uint16) *Builder                { b.mp = v; return b }
func (b *Builder) SetWeaponAttack(v uint16) *Builder      { b.weaponAttack = v; return b }
func (b *Builder) SetMagicAttack(v uint16) *Builder       { b.magicAttack = v; return b }
func (b *Builder) SetWeaponDefense(v uint16) *Builder     { b.weaponDefense = v; return b }
func (b *Builder) SetMagicDefense(v uint16) *Builder      { b.magicDefense = v; return b }
func (b *Builder) SetAccuracy(v uint16) *Builder          { b.accuracy = v; return b }
func (b *Builder) SetAvoidability(v uint16) *Builder      { b.avoidability = v; return b }
func (b *Builder) SetHands(v uint16) *Builder             { b.hands = v; return b }
func (b *Builder) SetSpeed(v uint16) *Builder             { b.speed = v; return b }
func (b *Builder) SetJump(v uint16) *Builder              { b.jump = v; return b }
func (b *Builder) SetSlots(v uint16) *Builder             { b.slots = v; return b }
func (b *Builder) SetLocked(v bool) *Builder {
	if v {
		b.flag = af.SetFlag(b.flag, af.FlagLock)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagLock)
	}
	return b
}

func (b *Builder) SetSpikes(v bool) *Builder {
	if v {
		b.flag = af.SetFlag(b.flag, af.FlagSpikes)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagSpikes)
	}
	return b
}

// SetKarmaUsed sets or clears the slot-class-correct karma bit, touching NO
// other bit. On an equip the bundle karma bit (0x02) is FlagSpikes, so a
// hand-picked constant here would render spikes on every karma'd equip.
func (b *Builder) SetKarmaUsed(v bool) *Builder {
	f, ok := af.KarmaFlagFor(b.templateId)
	if !ok {
		return b
	}
	if v {
		b.flag = af.SetFlag(b.flag, f)
	} else {
		b.flag = af.ClearFlag(b.flag, f)
	}
	return b
}

func (b *Builder) SetCold(v bool) *Builder {
	if v {
		b.flag = af.SetFlag(b.flag, af.FlagCold)
	} else {
		b.flag = af.ClearFlag(b.flag, af.FlagCold)
	}
	return b
}

func (b *Builder) SetCanBeTraded(v bool) *Builder {
	if v {
		b.flag = af.ClearFlag(b.flag, af.FlagUntradeable)
	} else {
		b.flag = af.SetFlag(b.flag, af.FlagUntradeable)
	}
	return b
}

func (b *Builder) AddFlag(f af.Flag) *Builder {
	b.flag = af.SetFlag(b.flag, f)
	return b
}

func (b *Builder) RemoveFlag(f af.Flag) *Builder {
	b.flag = af.ClearFlag(b.flag, f)
	return b
}
func (b *Builder) SetLevelType(v byte) *Builder        { b.levelType = v; return b }
func (b *Builder) SetLevel(v byte) *Builder            { b.level = v; return b }
func (b *Builder) SetExperience(v uint32) *Builder     { b.experience = v; return b }
func (b *Builder) SetHammersApplied(v uint32) *Builder { b.hammersApplied = v; return b }

func (b *Builder) SetEquippedSince(t *time.Time) *Builder { b.equippedSince = t; return b }
func (b *Builder) SetCashId(v int64) *Builder             { b.cashId = v; return b }
func (b *Builder) SetCommodityId(v uint32) *Builder       { b.commodityId = v; return b }
func (b *Builder) SetPurchaseBy(v uint32) *Builder        { b.purchaseBy = v; return b }
func (b *Builder) SetPetId(v uint32) *Builder             { b.petId = v; return b }

func (b *Builder) Build() Model {
	return Model{
		id:             b.id,
		compartmentId:  b.compartmentId,
		slot:           b.slot,
		templateId:     b.templateId,
		expiration:     b.expiration,
		createdAt:      b.createdAt,
		quantity:       b.quantity,
		ownerId:        b.ownerId,
		owner:          b.owner,
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
