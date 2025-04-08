package asset

import "time"

type Model[E any] struct {
	id            uint32
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType string
	referenceData E
}

func (m Model[E]) Id() uint32 {
	return m.id
}

func (m Model[E]) Slot() int16 {
	return m.slot
}

func (m Model[E]) TemplateId() uint32 {
	return m.templateId
}

func (m Model[E]) Expiration() time.Time {
	return m.expiration
}

func (m Model[E]) ReferenceId() uint32 {
	return m.referenceId
}

func (m Model[E]) ReferenceType() string {
	return m.referenceType
}

func Clone[E any](m Model[E]) *ModelBuilder[E] {
	return &ModelBuilder[E]{
		id:            m.id,
		slot:          m.slot,
		templateId:    m.templateId,
		expiration:    m.expiration,
		referenceId:   m.referenceId,
		referenceType: m.referenceType,
		referenceData: m.referenceData,
	}
}

type ModelBuilder[E any] struct {
	id            uint32
	slot          int16
	templateId    uint32
	expiration    time.Time
	referenceId   uint32
	referenceType string
	referenceData E
}

func NewBuilder[E any](id uint32, templateId uint32, referenceId uint32, referenceType string) *ModelBuilder[E] {
	return &ModelBuilder[E]{
		id:            id,
		slot:          0,
		templateId:    templateId,
		expiration:    time.Time{},
		referenceId:   referenceId,
		referenceType: referenceType,
	}
}

func (b *ModelBuilder[E]) SetSlot(slot int16) *ModelBuilder[E] {
	b.slot = slot
	return b
}

func (b *ModelBuilder[E]) SetExpiration(e time.Time) *ModelBuilder[E] {
	b.expiration = e
	return b
}

func (b *ModelBuilder[E]) SetReferenceData(e E) *ModelBuilder[E] {
	b.referenceData = e
	return b
}

func (b *ModelBuilder[E]) Build() Model[E] {
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

type ConsumableReferenceData struct {
	quantity     uint32
	owner        string
	flag         uint16
	rechargeable uint64
}

type SetupReferenceData struct {
	quantity uint32
	owner    string
	flag     uint16
}

type EtcReferenceData struct {
	quantity uint32
	owner    string
	flag     uint16
}

type CashReferenceData struct {
	cashId     uint64
	quantity   uint32
	owner      uint32
	flag       uint16
	purchaseBy uint32
}

type PetReferenceData struct {
	cashId        uint64
	owner         uint32
	flag          uint16
	purchaseBy    uint32
	name          string
	level         byte
	closeness     uint16
	fullness      byte
	expiration    time.Time
	attribute     uint16
	skill         uint16
	remainingLife uint32
	attribute2    uint16
}
