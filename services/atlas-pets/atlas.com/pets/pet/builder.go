package pet

import (
	"atlas-pets/pet/exclude"
	"errors"
	"time"
)

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

func (b *ModelBuilder) Build() (Model, error) {
	if b.templateId == 0 {
		return Model{}, errors.New("templateId is required")
	}
	if b.ownerId == 0 {
		return Model{}, errors.New("ownerId is required")
	}
	if b.name == "" {
		return Model{}, errors.New("name is required")
	}
	if b.level < 1 || b.level > 30 {
		return Model{}, errors.New("level must be between 1 and 30")
	}
	if b.fullness > 100 {
		return Model{}, errors.New("fullness must be between 0 and 100")
	}
	if b.slot < -1 || b.slot > 2 {
		return Model{}, errors.New("slot must be -1 or between 0 and 2")
	}

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
	}, nil
}
