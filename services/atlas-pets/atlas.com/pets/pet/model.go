package pet

import (
	"atlas-pets/pet/exclude"
	"time"
)

type Model struct {
	id         uint32
	cashId     uint64
	templateId uint32
	name       string
	level      byte
	closeness  uint16
	fullness   byte
	expiration time.Time
	ownerId    uint32
	slot       int8
	excludes   []exclude.Model
	flag       uint16
	purchaseBy uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) CashId() uint64 {
	return m.cashId
}

func (m Model) TemplateId() uint32 {
	return m.templateId
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) Closeness() uint16 {
	return m.closeness
}

func (m Model) Fullness() byte {
	return m.fullness
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}

func (m Model) Lead() bool {
	return m.Slot() == 0
}

func (m Model) Slot() int8 {
	return m.slot
}

func (m Model) SetSlot(slot int8) Model {
	return Clone(m).SetSlot(slot).Build()
}

func (m Model) Excludes() []exclude.Model {
	return m.excludes
}

func (m Model) Flag() uint16 {
	return m.flag
}

func (m Model) PurchaseBy() uint32 {
	return m.purchaseBy
}

type ModelBuilder struct {
	id         uint32
	cashId     uint64
	templateId uint32
	name       string
	level      byte
	closeness  uint16
	fullness   byte
	expiration time.Time
	ownerId    uint32
	slot       int8
	excludes   []exclude.Model
	flag       uint16
	purchaseBy uint32
}

func NewModelBuilder(id uint32, cashId uint64, templateId uint32, name string, ownerId uint32) *ModelBuilder {
	return &ModelBuilder{
		id:         id,
		cashId:     cashId,
		templateId: templateId,
		name:       name,
		level:      1,
		closeness:  0,
		fullness:   100,
		expiration: time.Now().Add(2160 * time.Hour),
		ownerId:    ownerId,
		slot:       -1,
		excludes:   make([]exclude.Model, 0),
		flag:       0,
		purchaseBy: ownerId,
	}
}

func Clone(m Model) *ModelBuilder {
	return NewModelBuilder(m.Id(), m.CashId(), m.TemplateId(), m.Name(), m.OwnerId()).
		SetLevel(m.Level()).
		SetCloseness(m.Closeness()).
		SetFullness(m.Fullness()).
		SetExpiration(m.Expiration()).
		SetSlot(m.Slot()).
		SetExcludes(m.Excludes()).
		SetFlag(m.Flag()).
		SetPurchaseBy(m.PurchaseBy())
}

func (b *ModelBuilder) SetLevel(level byte) *ModelBuilder {
	b.level = level
	return b
}

func (b *ModelBuilder) SetCloseness(closeness uint16) *ModelBuilder {
	b.closeness = closeness
	return b
}

func (b *ModelBuilder) SetFullness(fullness byte) *ModelBuilder {
	b.fullness = fullness
	return b
}

func (b *ModelBuilder) SetExpiration(expiration time.Time) *ModelBuilder {
	b.expiration = expiration
	return b
}

func (b *ModelBuilder) SetSlot(slot int8) *ModelBuilder {
	b.slot = slot
	return b
}

func (b *ModelBuilder) SetExcludes(excludes []exclude.Model) *ModelBuilder {
	b.excludes = excludes
	return b
}

func (b *ModelBuilder) SetFlag(flag uint16) *ModelBuilder {
	b.flag = flag
	return b
}

func (b *ModelBuilder) SetPurchaseBy(by uint32) *ModelBuilder {
	b.purchaseBy = by
	return b
}

// Build returns the constructed Model instance
func (b *ModelBuilder) Build() Model {
	return Model{
		id:         b.id,
		cashId:     b.cashId,
		templateId: b.templateId,
		name:       b.name,
		level:      b.level,
		closeness:  b.closeness,
		fullness:   b.fullness,
		expiration: b.expiration,
		ownerId:    b.ownerId,
		slot:       b.slot,
		excludes:   b.excludes,
		flag:       b.flag,
		purchaseBy: b.purchaseBy,
	}
}
